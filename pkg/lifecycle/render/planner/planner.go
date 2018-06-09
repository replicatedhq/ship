package planner

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedcom/ship/pkg/templates"
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

func NewPlanner(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	ui cli.Ui,
	builderBuilder *templates.BuilderBuilder,
	saver docker.ImageSaver,
	urlResolver docker.PullURLResolver,
) Planner {
	return &CLIPlanner{
		Logger:         logger,
		Fs:             fs,
		UI:             ui,
		Viper:          v,
		BuilderBuilder: builderBuilder,
		Saver:          saver,
		URLResolver:    urlResolver,
	}
}

func (p *CLIPlanner) WithDaemon(d config.Daemon) Planner {
	p.Daemon = d
	return p
}
