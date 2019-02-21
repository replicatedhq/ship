package unfork

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/util"
	yaml "gopkg.in/yaml.v2"
)

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func (l *Unforker) PreExecute(ctx context.Context, step api.Step) error {
	// Split multi doc forked base first as it will be unmarshalled incorrectly in the following steps
	if err := l.maybeSplitMultidocYaml(ctx, step.Unfork.ForkedBase); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml forked base")
	}

	// Split the forked list and only save this result to state to reconstruct the rendered
	if err := l.maybeSplitListYaml(ctx, step.Unfork.ForkedBase, true); err != nil {
		return errors.Wrap(err, "maybe split list yaml forked base")
	}

	if err := l.maybeSplitMultidocYaml(ctx, step.Unfork.UpstreamBase); err != nil {
		return errors.Wrap(err, "maybe split multi doc yaml upstream base")
	}

	if err := l.maybeSplitListYaml(ctx, step.Unfork.UpstreamBase, false); err != nil {
		return errors.Wrap(err, "maybe split list yaml upstream base")
	}

	return nil
}

func (l *Unforker) maybeSplitListYaml(ctx context.Context, path string, saveList bool) error {
	debug := level.Debug(log.With(l.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github"))

	debug.Log("event", "readDir", "path", path)
	files, err := l.FS.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", path)
	}

	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			return l.maybeSplitListYaml(ctx, filepath.Join(path, file.Name()), saveList)
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
			listItems := make([]util.MinimalK8sYaml, 0)
			for idx, item := range k8sYaml.Items {
				itemK8sYaml := util.MinimalK8sYaml{}
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

			list := util.List{
				APIVersion: k8sYaml.APIVersion,
				Path:       filePath,
				Items:      listItems,
			}

			if saveList {
				debug.Log("event", "serializeListsMetadata")
				if err := l.State.SerializeListsMetadata(list); err != nil {
					return errors.Wrapf(err, "serialize list metadata")
				}
			}
		}
	}

	return nil
}

// TODO(Robert): Unused
func (l *Unforker) initialKustomizeRun(ctx context.Context, step api.Unfork) error {
	if err := l.writeBase(step); err != nil {
		return errors.Wrap(err, "write base kustomization")
	}

	built, err := l.unforkBuild(step.UpstreamBase)
	if err != nil {
		return errors.Wrap(err, "build overlay")
	}

	if err := l.writePostKustomizeFiles(step, built); err != nil {
		return errors.Wrap(err, "write initial kustomized yaml")
	}

	if err := l.replaceOriginal(step, built); err != nil {
		return errors.Wrap(err, "replace original yaml")
	}

	return nil
}

func (l *Unforker) replaceOriginal(step api.Unfork, built []util.PostKustomizeFile) error {
	builtMap := make(map[util.MinimalK8sYaml]util.PostKustomizeFile)
	for _, builtFile := range built {
		builtMap[builtFile.Minimal] = builtFile
	}

	if err := l.FS.Walk(step.UpstreamBase, func(targetPath string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed to walk base path")
		}

		if !l.shouldAddFileToBase(step.UpstreamBase, []string{}, targetPath) {
			if strings.HasSuffix(targetPath, "kustomization.yaml") {
				if err := l.FS.Remove(targetPath); err != nil {
					return errors.Wrap(err, "remove kustomization yaml")
				}
			}

			return nil
		}

		originalFileB, err := l.FS.ReadFile(targetPath)
		if err != nil {
			return errors.Wrap(err, "read original file")
		}

		originalMinimal := util.MinimalK8sYaml{}
		if err := yaml.Unmarshal(originalFileB, &originalMinimal); err != nil {
			return errors.Wrap(err, "unmarshal original")
		}

		if originalMinimal.Kind == "CustomResourceDefinition" {
			// Skip CRDs
			return nil
		}

		initKustomized, exists := builtMap[originalMinimal]
		if !exists {
			// Skip if the file does not have a kustomized equivalent
			return nil
		}

		if err := l.FS.Remove(targetPath); err != nil {
			return errors.Wrap(err, "remove original file")
		}

		initKustomizedB, err := yaml.Marshal(initKustomized.Full)
		if err != nil {
			return errors.Wrap(err, "marshal init kustomized")
		}

		if err := l.FS.WriteFile(targetPath, initKustomizedB, info.Mode()); err != nil {
			return errors.Wrap(err, "write init kustomized file")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "replace original with init kustomized")
	}

	return nil
}
