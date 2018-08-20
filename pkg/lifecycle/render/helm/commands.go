package helm

import (
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helm"
)

// Commands are Helm commands that are available to the Ship binary.
type Commands interface {
	Init() error
	DependencyUpdate(chartRoot string) error
	Template(chartName string, args []string) error
}

type helmCommands struct {
	Home string
}

func (h *helmCommands) Init() error {
	_, err := helm.Init(constants.TempHelmHomePath)
	return err
}

func (h *helmCommands) DependencyUpdate(chartRoot string) error {
	dependencyArgs := []string{
		"update",
		chartRoot,
	}
	dependencyCommand, err := helm.NewDependencyCmd(dependencyArgs)
	if err != nil {
		return err
	}
	return dependencyCommand.Execute()
}

func (h *helmCommands) Template(chartName string, args []string) error {
	templateCommand, err := helm.NewTemplateCmd(append([]string{chartName}, args...))
	if err != nil {
		return err
	}
	return templateCommand.Execute()
}

// NewCommands returns a helmCommands struct that implements Commands.
func NewCommands() Commands {
	return &helmCommands{}
}
