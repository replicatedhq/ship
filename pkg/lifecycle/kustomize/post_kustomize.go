package kustomize

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/state"
	yaml "gopkg.in/yaml.v2"
)

type postKustomizeFile struct {
	minimal state.MinimalK8sYaml
	full    interface{}
}

type postKustomizeCollection []postKustomizeFile

func (p postKustomizeCollection) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p postKustomizeCollection) Less(i, j int) bool {
	postKustomizeFileI := p[i]
	postKustomizeFileJ := p[j]

	if postKustomizeFileI.minimal.Kind == "List" {
		if postKustomizeFileJ.minimal.Kind != "List" {
			return true
		}
	}

	if postKustomizeFileI.minimal.Metadata.Namespace == postKustomizeFileJ.minimal.Metadata.Namespace {
		return postKustomizeFileI.minimal.Metadata.Name < postKustomizeFileJ.minimal.Metadata.Name
	}

	return postKustomizeFileI.minimal.Metadata.Namespace < postKustomizeFileJ.minimal.Metadata.Namespace
}

func (p postKustomizeCollection) Len() int {
	return len(p)
}

func (l *Kustomizer) rebuildListYaml(lists []state.List, kustomizedYamlFiles []postKustomizeFile) ([]postKustomizeFile, error) {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "rebuildListYaml"))
	yamlMap := make(map[state.MinimalK8sYaml]interface{})

	for _, postKustomizeFile := range kustomizedYamlFiles {
		yamlMap[postKustomizeFile.minimal] = postKustomizeFile.full
	}

	fullReconstructedRendered := make([]postKustomizeFile, 0)
	for _, list := range lists {
		var allListItems []interface{}
		for _, item := range list.Items {
			if full, exists := yamlMap[item]; exists {
				delete(yamlMap, item)
				allListItems = append(allListItems, full)
			}
		}

		reconstructedList := specs.ListK8sYaml{
			APIVersion: list.APIVersion,
			Kind:       "List",
			Items:      allListItems,
		}

		debug.Log("event", "marshal reconstructed")
		postKustomizeList := postKustomizeFile{
			minimal: state.MinimalK8sYaml{
				Kind: "List",
			},
			full: reconstructedList,
		}

		fullReconstructedRendered = append(fullReconstructedRendered, postKustomizeList)
	}

	for nonListYamlMinimal, nonListYamlFull := range yamlMap {
		fullReconstructedRendered = append(fullReconstructedRendered, postKustomizeFile{
			minimal: nonListYamlMinimal,
			full:    nonListYamlFull,
		})
	}

	return fullReconstructedRendered, nil
}

func (l *Kustomizer) writePostKustomizeFiles(step api.Kustomize, postKustomizeFiles []postKustomizeFile) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "writePostKustomizeFiles"))

	var joinedFinal string
	for _, file := range postKustomizeFiles {
		debug.Log("event", "marshal post kustomize file")
		fileB, err := yaml.Marshal(file.full)
		if err != nil {
			return errors.Wrapf(err, "marshal file %s", file.minimal.Metadata.Name)
		}

		if joinedFinal != "" {
			joinedFinal += "\n---\n" + string(fileB)
		} else {
			joinedFinal += string(fileB)
		}
	}

	debug.Log("event", "write post kustomize files", "dest", step.Dest)
	if err := l.FS.WriteFile(step.Dest, []byte(joinedFinal), os.FileMode(0644)); err != nil {
		return errors.Wrapf(err, "write kustomized and post processed yaml at %s", step.Dest)
	}

	return nil
}
