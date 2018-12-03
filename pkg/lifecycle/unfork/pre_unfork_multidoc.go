package unfork

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/util"
	yaml "gopkg.in/yaml.v2"
)

// this function is not perfect, and has known limitations. One of these is that it does not account for `\n---\n` in multiline strings.
func (l *Unforker) maybeSplitMultidocYaml(ctx context.Context, localPath string) error {
	type outputYaml struct {
		name     string
		contents string
	}

	files, err := l.FS.ReadDir(localPath)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", localPath)
	}

	for _, file := range files {
		if file.IsDir() {
			if err := l.maybeSplitMultidocYaml(ctx, filepath.Join(localPath, file.Name())); err != nil {
				return err
			}
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			continue
		}

		inFileBytes, err := l.FS.ReadFile(filepath.Join(localPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "read %s", filepath.Join(localPath, file.Name()))
		}

		outputFiles := []outputYaml{}
		filesStrings := strings.Split(string(inFileBytes), "\n---\n")

		// generate replacement yaml files
		for idx, fileString := range filesStrings {

			thisOutputFile := outputYaml{contents: fileString}

			thisMetadata := util.MinimalK8sYaml{}
			_ = yaml.Unmarshal([]byte(fileString), &thisMetadata)

			if thisMetadata.Kind == "" || thisMetadata.Kind == "CustomResourceDefinition" {
				// ignore invalid k8s yaml and CRDs
				continue
			}

			fileName := util.GenerateNameFromMetadata(thisMetadata, idx)
			thisOutputFile.name = fileName
			outputFiles = append(outputFiles, thisOutputFile)
		}

		if len(outputFiles) < 2 {
			// not a multidoc yaml, or at least not a multidoc kubernetes yaml
			continue
		}

		// delete multidoc yaml file
		err = l.FS.Remove(filepath.Join(localPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "unable to remove %s", filepath.Join(localPath, file.Name()))
		}

		// write replacement yaml
		for _, outputFile := range outputFiles {
			err = l.FS.WriteFile(filepath.Join(localPath, outputFile.name+".yaml"), []byte(outputFile.contents), os.FileMode(0644))
			if err != nil {
				return errors.Wrapf(err, "write %s", outputFile.name)
			}
		}
	}

	return nil
}
