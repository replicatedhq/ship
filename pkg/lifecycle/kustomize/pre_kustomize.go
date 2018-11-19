package kustomize

import (
	"context"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	yaml "gopkg.in/yaml.v2"
)

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func (l *Kustomizer) PreExecute(ctx context.Context, step api.Step) error {
	return l.maybeSplitListYaml(ctx, step.Kustomize.Base)
}

func (l *Kustomizer) maybeSplitListYaml(ctx context.Context, path string) error {
	debug := level.Debug(log.With(l.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github"))

	debug.Log("event", "readDir", "path", path)
	files, err := l.FS.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", path)
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			return l.maybeSplitListYaml(ctx, filepath.Join(path, file.Name()))
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			return nil
		}

		fileB, err := l.FS.ReadFile(filePath)
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

				fileName := util.GenerateNameFromMetadata(itemK8sYaml, idx)
				if err := l.FS.WriteFile(filepath.Join(path, fileName+".yaml"), []byte(itemB), os.FileMode(0644)); err != nil {
					return errors.Wrap(err, "write yaml")
				}

				listItems = append(listItems, itemK8sYaml)
			}

			if err := l.FS.Remove(filePath); err != nil {
				return errors.Wrapf(err, "remove k8s list %s", filePath)
			}

			list := state.List{
				APIVersion: k8sYaml.APIVersion,
				Path:       filePath,
				Items:      listItems,
			}

			debug.Log("event", "serializeListsMetadata")
			if err := l.State.SerializeListsMetadata(list); err != nil {
				return errors.Wrapf(err, "serialize list metadata")
			}
		}
	}

	return nil
}
