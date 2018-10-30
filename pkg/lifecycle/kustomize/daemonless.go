package kustomize

import (
	"context"
	"os"
	"strings"

	"github.com/replicatedhq/ship/pkg/specs"
	yaml "gopkg.in/yaml.v2"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type Kustomizer struct {
	Logger  log.Logger
	FS      afero.Afero
	State   state.Manager
	Patcher patch.ShipPatcher
}

func NewDaemonlessKustomizer(
	logger log.Logger,
	fs afero.Afero,
	state state.Manager,
) lifecycle.Kustomizer {
	return &Kustomizer{
		Logger: logger,
		FS:     fs,
		State:  state,
	}
}

func (l *Kustomizer) Execute(ctx context.Context, release *api.Release, step api.Kustomize) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "execute"))

	debug.Log("event", "write.base.kustomization.yaml")
	err := l.writeBase(step)
	if err != nil {
		return errors.Wrap(err, "write base kustomization")
	}

	current, err := l.State.TryLoad()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	debug.Log("event", "state.loaded")
	kustomizeState := current.CurrentKustomize()

	var shipOverlay state.Overlay
	if kustomizeState == nil {
		debug.Log("event", "state.kustomize.empty")
	} else {
		shipOverlay = kustomizeState.Ship()
	}

	fs, err := l.getPotentiallyChrootedFs(release)
	if err != nil {
		debug.Log("event", "getFs.fail")
		return errors.Wrapf(err, "get base fs")
	}

	debug.Log("event", "mkdir", "dir", step.OverlayPath())
	err = fs.MkdirAll(step.OverlayPath(), 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", step.OverlayPath())
		return errors.Wrapf(err, "make dir %s", step.OverlayPath())
	}

	relativePatchPaths, err := l.writePatches(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	relativeResourcePaths, err := l.writeResources(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	err = l.writeOverlay(fs, step, relativePatchPaths, relativeResourcePaths)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

	if step.Dest != "" {
		debug.Log("event", "kustomize.build", "dest", step.Dest)
		err = l.kustomizeBuild(fs, step)
		if err != nil {
			return errors.Wrap(err, "build overlay")
		}

		debug.Log("event", "try load state")
		currentState, err := l.State.TryLoad()
		if err != nil {
			return errors.Wrap(err, "try load state")
		}

		if currentState.Versioned().V1.Metadata != nil {
			lists := currentState.Versioned().V1.Metadata.Lists
			if len(lists) > 0 {

				debug.Log("event", "kustomize.rebuildListYaml")
				if err := l.rebuildListYaml(step, lists); err != nil {
					return errors.Wrap(err, "rebuild list yaml")
				}
			}
		}
	}

	return nil
}
func (l *Kustomizer) kustomizeBuild(fs afero.Afero, kustomize api.Kustomize) error {
	builtYAML, err := l.Patcher.RunKustomize(kustomize.OverlayPath())
	if err != nil {
		return errors.Wrap(err, "run kustomize")
	}

	fs.WriteFile(kustomize.Dest, builtYAML, 0644)
	return nil
}

func (l *Kustomizer) rebuildListYaml(kustomize api.Kustomize, lists []state.List) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "rebuildListYaml"))
	yamlMap := make(map[state.MinimalK8sYaml]interface{})

	debug.Log("event", "read rendered yaml", "dest", kustomize.Dest)
	renderedB, err := l.FS.ReadFile(kustomize.Dest)
	if err != nil {
		return errors.Wrap(err, "read kustomize rendered yaml")
	}

	files := strings.Split(string(renderedB), "\n---\n")
	for _, file := range files {
		var fullYaml interface{}

		debug.Log("event", "unmarshal part of rendered")
		if err := yaml.Unmarshal([]byte(file), &fullYaml); err != nil {
			return errors.Wrap(err, "unmarshal part of rendered")
		}

		debug.Log("event", "unmarshal part of rendered to minimal")
		minimal := state.MinimalK8sYaml{}
		if err := yaml.Unmarshal([]byte(file), &minimal); err != nil {
			return errors.Wrap(err, "unmarshal part of rendered to minimal")
		}

		yamlMap[minimal] = fullYaml
	}

	var fullReconstructedRendered string
	for _, list := range lists {
		var allListItems []interface{}
		for _, item := range list.Items {
			if full, exists := yamlMap[item]; exists {
				delete(yamlMap, item)
				allListItems = append(allListItems, full)
			}
		}

		reconstructed := specs.ListK8sYaml{
			APIVersion: list.APIVersion,
			Kind:       "List",
			Items:      allListItems,
		}

		debug.Log("event", "marshal reconstructed")
		reconstructedB, err := yaml.Marshal(reconstructed)
		if err != nil {
			return errors.Wrapf(err, "marshal reconstructed yaml %s", list.Path)
		}

		if fullReconstructedRendered != "" {
			fullReconstructedRendered += "\n---\n"
		}

		fullReconstructedRendered += string(reconstructedB)
	}

	for _, nonListYaml := range yamlMap {
		nonListYamlB, err := yaml.Marshal(nonListYaml)
		if err != nil {
			return errors.Wrapf(err, "marshal non list yaml")
		}

		if fullReconstructedRendered != "" {
			fullReconstructedRendered += "\n---\n"
		}

		fullReconstructedRendered += string(nonListYamlB)
	}

	debug.Log("event", "write reconstructed", "dest", kustomize.Dest)
	if err := l.FS.WriteFile(kustomize.Dest, []byte(fullReconstructedRendered), os.FileMode(0644)); err != nil {
		return errors.Wrapf(err, "write reconstructed dest %s", kustomize.Dest)
	}

	return nil
}
