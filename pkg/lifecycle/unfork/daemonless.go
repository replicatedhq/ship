package unfork

import (
	"context"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

type Unforker struct {
	Logger  log.Logger
	FS      afero.Afero
	State   state.Manager
	Patcher patch.Patcher
}

func NewDaemonlessUnforker(logger log.Logger, fs afero.Afero, state state.Manager, patcher patch.Patcher) lifecycle.Unforker {
	return &Unforker{
		Logger:  logger,
		FS:      fs,
		State:   state,
		Patcher: patcher,
	}
}

func (l *Unforker) Execute(ctx context.Context, release *api.Release, step api.Unfork) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.unforker", "method", "execute"))

	debug.Log("event", "write.base.kustomization.yaml")
	err := l.writeBase(step)
	if err != nil {
		return errors.Wrap(err, "write base kustomization")
	}

	debug.Log("event", "state.loaded")
	fs, err := l.getPotentiallyChrootedFs(release)
	if err != nil {
		debug.Log("event", "getFs.fail")
		return errors.Wrapf(err, "get base fs")
	}

	debug.Log("event", "mkdir", "dir", step.OverlayPath())
	err = l.FS.MkdirAll(step.OverlayPath(), 0777)
	if err != nil {
		debug.Log("event", "mkdir.fail", "dir", step.OverlayPath())
		return errors.Wrapf(err, "make dir %s", step.OverlayPath())
	}

	// this isn't in the right place, but it works until we figure out the right workflow...
	kustomizeState, err := l.generatePatches(fs, step)
	if err != nil {
		debug.Log("event", "generate.patches.fail", "err", err)
		return errors.Wrapf(err, "generate patches")
	}

	shipOverlay := kustomizeState.Ship()

	relativePatchPaths, err := l.writePatches(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	relativeResourcePaths, err := l.writeResources(fs, shipOverlay, step.OverlayPath())
	if err != nil {
		return err
	}

	err = l.writeOverlay(step, relativePatchPaths, relativeResourcePaths)
	if err != nil {
		return errors.Wrap(err, "write overlay")
	}

	if step.Dest != "" {
		debug.Log("event", "unfork.build", "dest", step.Dest)
		built, err := l.unforkBuild(step.OverlayPath())
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
				debug.Log("event", "unfork.rebuildListYaml")
				if built, err = l.rebuildListYaml(lists, built); err != nil {
					return errors.Wrap(err, "rebuild list yaml")
				}
			}
		}

		if err := l.writePostKustomizeFiles(step, built); err != nil {
			return errors.Wrapf(err, "write kustomized and post processed yaml at %s", step.Dest)
		}
	}

	return nil
}

func (l *Unforker) unforkBuild(kustomizePath string) ([]postKustomizeFile, error) {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.unforker", "method", "kustomizeBuild"))

	builtYAML, err := l.Patcher.RunKustomize(kustomizePath)
	if err != nil {
		return nil, errors.Wrap(err, "run kustomize")
	}

	files := strings.Split(string(builtYAML), "\n---\n")
	postKustomizeFiles := make([]postKustomizeFile, 0)
	for idx, file := range files {
		var fullYaml interface{}

		debug.Log("event", "unmarshal part of rendered")
		if err := yaml.Unmarshal([]byte(file), &fullYaml); err != nil {
			return postKustomizeFiles, errors.Wrap(err, "unmarshal part of rendered")
		}

		debug.Log("event", "unmarshal part of rendered to minimal")
		minimal := util.MinimalK8sYaml{}
		if err := yaml.Unmarshal([]byte(file), &minimal); err != nil {
			return postKustomizeFiles, errors.Wrap(err, "unmarshal part of rendered to minimal")
		}

		postKustomizeFiles = append(postKustomizeFiles, postKustomizeFile{
			order:   idx,
			minimal: minimal,
			full:    fullYaml,
		})
	}

	return postKustomizeFiles, nil
}
