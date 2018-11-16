package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	yaml "gopkg.in/yaml.v2"
)

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func (r *LocalRenderer) maybeSplitListYaml(ctx context.Context, asset api.GitHubAsset, builder *templates.Builder) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github"))
	assetDestPath, err := getDestPathNoProxy(asset, builder)
	if err != nil {
		return errors.Wrap(err, "get dest path")
	}
	path := filepath.Dir(assetDestPath)

	debug.Log("event", "readDir", "path", path)
	files, err := r.Fs.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", path)
	}

	var lists []state.List
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			continue
			// TODO: handling nested list yamls
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			return nil
		}

		fileB, err := r.Fs.ReadFile(filePath)
		if err != nil {
			return errors.Wrapf(err, "read %s", filePath)
		}

		k8sYaml := ListK8sYaml{}
		if err := yaml.Unmarshal(fileB, &k8sYaml); err != nil {
			return errors.Wrapf(err, "unmarshal %s", filePath)
		}

		if k8sYaml.Kind == "List" {
			listItems := make([]state.MinimalK8sYaml, 0)
			for idx, item := range k8sYaml.Items {
				itemK8sYaml := state.MinimalK8sYaml{}
				itemB, err := yaml.Marshal(item)
				if err != nil {
					return errors.Wrapf(err, "marshal item %d from %s", idx, filePath)
				}

				if err := yaml.Unmarshal(itemB, &itemK8sYaml); err != nil {
					return errors.Wrap(err, "unmarshal item")
				}

				fileName := generateNameFromMetadata(itemK8sYaml, idx)
				if err := r.Fs.WriteFile(filepath.Join(path, fileName+".yaml"), []byte(itemB), os.FileMode(0644)); err != nil {
					return errors.Wrap(err, "write yaml")
				}

				listItems = append(listItems, itemK8sYaml)
			}
			list := state.List{
				APIVersion: k8sYaml.APIVersion,
				Path:       filePath,
				Items:      listItems,
			}
			lists = append(lists, list)

			if err := r.Fs.Remove(filePath); err != nil {
				return errors.Wrapf(err, "remove k8s list %s", filePath)
			}
		}
	}

	if err := r.StateManager.SerializeListsMetadata(lists); err != nil {
		return errors.Wrapf(err, "serialize list metadata")
	}

	return nil
}

func generateNameFromMetadata(k8sYaml state.MinimalK8sYaml, idx int) string {
	fileName := fmt.Sprintf("%s-%d", k8sYaml.Kind, idx)

	if k8sYaml.Metadata.Name != "" {
		fileName = k8sYaml.Kind + "-" + k8sYaml.Metadata.Name
		if k8sYaml.Metadata.Namespace != "" && k8sYaml.Metadata.Namespace != "default" {
			fileName += "-" + k8sYaml.Metadata.Namespace
		}
	}

	return fileName
}
