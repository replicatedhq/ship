package helm

import (
	"os"

	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helm"
)

// Commands are Helm commands that are available to the Ship binary.
type Commands interface {
	Init() error
	DependencyUpdate(chartRoot string) error
	Template(chartName string, args []string) error
	Fetch(chartRef, repoURL, version, dest, home string) error
}

type helmCommands struct {
}

func (h *helmCommands) Fetch(chartRef, repoURL, version, dest, home string) error {
	outstring, err := helm.Fetch(chartRef, repoURL, version, dest, home)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("helm fetch failed, output %q", outstring))
	}
	return nil
}

func (h *helmCommands) Init() error {
	err := os.MkdirAll(constants.InternalTempHelmHome, 0755)
	if err != nil {
		return errors.Wrapf(err, "create %s", constants.InternalTempHelmHome)
	}
	output, err := helm.Init(constants.InternalTempHelmHome)
	return errors.Wrapf(err, "helm init: %s", output)
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
