package helm

import (
	"github.com/replicatedhq/ship/pkg/helm"
)

// Commands are Helm commands that are available to the Ship binary.
type Commands interface {
	Template(chartName string, args []string) error
}

type helmCommands struct{}

func (h *helmCommands) Template(chartName string, args []string) error {
	templateCommand := helm.NewTemplateCmd(append([]string{chartName}, args...))
	return templateCommand.Execute()
}

// NewCommands returns a helmCommands struct that implements Commands.
func NewCommands() Commands {
	return &helmCommands{}
}
