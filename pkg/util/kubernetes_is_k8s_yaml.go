package util

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

func ShouldAddFileToBase(fs *afero.Afero, excludedBases []string, targetPath string) bool {
	if filepath.Ext(targetPath) != ".yaml" && filepath.Ext(targetPath) != ".yml" {
		return false
	}

	for _, base := range excludedBases {
		basePathWOLeading := strings.TrimPrefix(base, "/")
		if basePathWOLeading == targetPath {
			return false
		}
	}

	if !IsK8sYaml(fs, targetPath) {
		return false
	}

	return !strings.HasSuffix(targetPath, "kustomization.yaml") &&
		!strings.HasSuffix(targetPath, "Chart.yaml") &&
		!strings.HasSuffix(targetPath, "values.yaml")
}

func IsK8sYaml(fs *afero.Afero, target string) bool {
	fileContents, err := fs.ReadFile(target)
	if err != nil {
		// if we cannot read a file, we assume that it is valid k8s yaml
		return true
	}

	originalMinimal := MinimalK8sYaml{}
	if err := yaml.Unmarshal(fileContents, &originalMinimal); err != nil {
		// if we cannot unmarshal the file, it is not valid k8s yaml
		return false
	}

	if originalMinimal.Kind == "" {
		// if there is not a kind, it is not valid k8s yaml
		return false
	}

	// k8s yaml must have a name OR be a list type
	return originalMinimal.Metadata.Name != "" || strings.HasSuffix(originalMinimal.Kind, "List")
}
