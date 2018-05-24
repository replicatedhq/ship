package planner

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/fs"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
)

// A Plan is a list of PlanSteps to execute
type Plan []Step

// A Step describes a single unit of work that Ship will do
// to render the application
type Step struct {
	Description string `json:"description" yaml:"description" hcl:"description"`
	Dest        string `json:"dest" yaml:"dest" hcl:"dest"`
	Execute     func(ctx context.Context) error
	Err         error
}

// Planner is a thing that can plan and execute rendering
type Planner interface {
	Build(
		[]api.Asset,
		[]libyaml.ConfigGroup,
		api.ReleaseMetadata,
		map[string]interface{},
	) Plan

	Confirm(Plan) (bool, error)
	Execute(context.Context, Plan) error
	WithDaemon(d config.Daemon) Planner
}

// CLIPlanner is the default Planner
type CLIPlanner struct {
	Logger         log.Logger
	Fs             afero.Afero
	UI             cli.Ui
	Viper          *viper.Viper
	Daemon         config.Daemon
	BuilderBuilder *templates.BuilderBuilder
	Saver          docker.ImageSaver
	URLResolver    docker.PullURLResolver
}

func FromViper(v *viper.Viper) (Planner, error) {
	saver, err := docker.SaverFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "initialize docker saver")
	}

	return &CLIPlanner{
		Logger:         logger.FromViper(v),
		Fs:             fs.FromViper(v),
		UI:             ui.FromViper(v),
		Viper:          v,
		BuilderBuilder: templates.BuilderBuilderFromViper(v),
		Saver:          saver,
		URLResolver:    docker.URLResolverFromViper(v),
	}, nil
}

func (p *CLIPlanner) WithDaemon(d config.Daemon) Planner {
	p.Daemon = d
	return p
}
