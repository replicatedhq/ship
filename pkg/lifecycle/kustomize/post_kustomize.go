package kustomize

import (
	"os"
	"sort"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
	"github.com/replicatedhq/ship/pkg/state"
	yaml "gopkg.in/yaml.v2"
)

type postKustomizeFile struct {
	order   int
	minimal state.MinimalK8sYaml
	full    interface{}
}

type postKustomizeFileCollection []postKustomizeFile

func (c postKustomizeFileCollection) Len() int {
	return len(c)
}

func (c postKustomizeFileCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c postKustomizeFileCollection) Less(i, j int) bool {
	return c[i].order < c[j].order
}

func (l *Kustomizer) rebuildListYaml(lists []state.List, kustomizedYamlFiles []postKustomizeFile) ([]postKustomizeFile, error) {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "rebuildListYaml"))
	yamlMap := make(map[state.MinimalK8sYaml]postKustomizeFile)

	for _, postKustomizeFile := range kustomizedYamlFiles {
		yamlMap[postKustomizeFile.minimal] = postKustomizeFile
	}

	fullReconstructedRendered := make([]postKustomizeFile, 0)
	for _, list := range lists {
		var allListItems []interface{}
		for _, item := range list.Items {
			if pkFile, exists := yamlMap[item]; exists {
				delete(yamlMap, item)
				allListItems = append(allListItems, pkFile.full)
			}
		}

		debug.Log("event", "reconstruct list")
		reconstructedList := github.ListK8sYaml{
			APIVersion: list.APIVersion,
			Kind:       "List",
			Items:      allListItems,
		}

		postKustomizeList := postKustomizeFile{
			minimal: state.MinimalK8sYaml{
				Kind: "List",
			},
			full: reconstructedList,
		}

		fullReconstructedRendered = append(fullReconstructedRendered, postKustomizeList)
	}

	for nonListYamlMinimal, pkFile := range yamlMap {
		fullReconstructedRendered = append(fullReconstructedRendered, postKustomizeFile{
			order:   pkFile.order,
			minimal: nonListYamlMinimal,
			full:    pkFile.full,
		})
	}

	return fullReconstructedRendered, nil
}

func (l *Kustomizer) writePostKustomizeFiles(step api.Kustomize, postKustomizeFiles []postKustomizeFile) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "writePostKustomizeFiles"))

	sort.Stable(postKustomizeFileCollection(postKustomizeFiles))

	var joinedFinal string
	for _, file := range postKustomizeFiles {
		debug.Log("event", "marshal post kustomize file")
		fileB, err := yaml.Marshal(file.full)
		if err != nil {
			return errors.Wrapf(err, "marshal file %s", file.minimal.Metadata.Name)
		}

		if joinedFinal != "" {
			joinedFinal += "---\n" + string(fileB)
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
