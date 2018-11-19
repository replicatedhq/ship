package util

import (
	"fmt"

	"github.com/replicatedhq/ship/pkg/state"
)

func GenerateNameFromMetadata(k8sYaml state.MinimalK8sYaml, idx int) string {
	fileName := fmt.Sprintf("%s-%d", k8sYaml.Kind, idx)

	if k8sYaml.Metadata.Name != "" {
		fileName = k8sYaml.Kind + "-" + k8sYaml.Metadata.Name
		if k8sYaml.Metadata.Namespace != "" && k8sYaml.Metadata.Namespace != "default" {
			fileName += "-" + k8sYaml.Metadata.Namespace
		}
	}

	return fileName
}
