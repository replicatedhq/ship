package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helm"
	"github.com/spf13/afero"
	"k8s.io/helm/pkg/chartutil"
)

// Commands are Helm commands that are available to the Ship binary.
type Commands interface {
	Init() error
	MaybeDependencyUpdate(chartRoot string) error
	Template(chartName string, args []string) error
	Fetch(chartRef, repoURL, version, dest, home string) error
	RepoAdd(name, url, home string) error
}

type helmCommands struct {
	logger log.Logger
	fs     afero.Afero
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

func (h *helmCommands) MaybeDependencyUpdate(chartRoot string) error {
	debug := level.Debug(log.With(h.logger, "method", "maybeDependencyUpdate"))

	requirementsExists, err := h.fs.Exists(filepath.Join(chartRoot, "requirements.yaml"))
	if err != nil {
		return errors.Wrap(err, "check requirements yaml existence")
	}

	if !requirementsExists {
		return nil
	}

	requirementsB, err := h.fs.ReadFile(filepath.Join(chartRoot, "requirements.yaml"))
	if err != nil {
		return errors.Wrap(err, "read requirements yaml")
	}

	requirements := chartutil.Requirements{}
	if err := yaml.Unmarshal(requirementsB, &requirements); err != nil {
		return errors.Wrap(err, "unmarshal requirements yaml")
	}

	allEmpty := true
	for _, dependency := range requirements.Dependencies {
		if dependency.Repository != "" {
			allEmpty = false
		}
	}

	if !allEmpty {
		debug.Log("event", "dependency update")
		if err := h.dependencyUpdate(chartRoot); err != nil {
			return errors.Wrap(err, "dependency update")
		}
	} else {
		debug.Log("event", "skip dependency update")
	}

	return nil
}

func (h *helmCommands) Template(chartName string, args []string) error {
	templateCommand, err := helm.NewTemplateCmd(append([]string{chartName}, args...))
	if err != nil {
		return err
	}
	return templateCommand.Execute()
}

// NewCommands returns a helmCommands struct that implements Commands.
func NewCommands(
	fs afero.Afero,
	logger log.Logger,
) Commands {
	return &helmCommands{
		logger: logger,
		fs:     fs,
	}
}

func (h *helmCommands) dependencyUpdate(chartRoot string) error {
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

func (h *helmCommands) RepoAdd(name, url, home string) error {
	outstring, err := helm.RepoAdd(name, url, home)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("helm repo add failed, output %q", outstring))
	}
	return nil
}
