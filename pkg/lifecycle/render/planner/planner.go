package planner

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/amazoneks"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/azureaks"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/dockerlayer"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/googlegke"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/helm"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/local"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/terraform"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/web"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// A Plan is a list of PlanSteps to execute
type Plan []Step

// A Execute describes a single unit of work that Ship will do
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
		string,
		[]api.Asset,
		[]libyaml.ConfigGroup,
		api.ReleaseMetadata,
		map[string]interface{},
	) (Plan, error)

	Execute(context.Context, Plan) error

	WithStatusReceiver(receiver daemontypes.StatusReceiver) Planner
}

type Factory func() *CLIPlanner

func (f Factory) WithStatusReceiver(receiver daemontypes.StatusReceiver) Planner {
	return Factory(func() *CLIPlanner {
		planner := f()
		return &CLIPlanner{
			Status: receiver,

			Logger:         planner.Logger,
			Fs:             planner.Fs,
			UI:             planner.UI,
			Viper:          planner.Viper,
			BuilderBuilder: planner.BuilderBuilder,

			Inline:      planner.Inline,
			Helm:        planner.Helm,
			Docker:      planner.Docker,
			DockerLayer: planner.DockerLayer,
			GitHub:      planner.GitHub,
			Terraform:   planner.Terraform,
			Web:         planner.Web,
			AmazonEKS:   planner.AmazonEKS,
			GoogleGKE:   planner.GoogleGKE,
			AzureAKS:    planner.AzureAKS,
		}

	})
}

func (f Factory) Build(
	root string,
	assets []api.Asset,
	configGroups []libyaml.ConfigGroup,
	releaseMeta api.ReleaseMetadata,
	templateContext map[string]interface{},
) (Plan, error) {
	planner := f()
	return planner.Build(root, assets, configGroups, releaseMeta, templateContext)
}

func (f Factory) Execute(ctx context.Context, p Plan) error {
	planner := f()
	return planner.Execute(ctx, p)
}

// CLIPlanner is the default Planner
type CLIPlanner struct {
	Logger         log.Logger
	Fs             afero.Afero
	UI             cli.Ui
	Viper          *viper.Viper
	Status         daemontypes.StatusReceiver
	BuilderBuilder *templates.BuilderBuilder

	Inline      inline.Renderer
	Helm        helm.Renderer
	Local       local.Renderer
	Docker      docker.Renderer
	DockerLayer *dockerlayer.Unpacker
	Web         web.Renderer
	GitHub      github.Renderer
	Terraform   terraform.Renderer
	AmazonEKS   amazoneks.Renderer
	GoogleGKE   googlegke.Renderer
	AzureAKS    azureaks.Renderer
}

// Use a factory so we can create instances and override the StatusReceiver on those instances.
func NewFactory(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	ui cli.Ui,
	builderBuilder *templates.BuilderBuilder,
	inlineRenderer inline.Renderer,
	dockerRenderer docker.Renderer,
	helmRenderer helm.Renderer,
	localRenderer local.Renderer,
	dockerlayers *dockerlayer.Unpacker,
	gh github.Renderer,
	tf terraform.Renderer,
	webRenderer web.Renderer,
	amazonEKS amazoneks.Renderer,
	googleGKE googlegke.Renderer,
	azureAKS azureaks.Renderer,
	status daemontypes.StatusReceiver,
) Planner {
	return Factory(func() *CLIPlanner {
		return &CLIPlanner{
			Logger:         logger,
			Fs:             fs,
			UI:             ui,
			Viper:          v,
			BuilderBuilder: builderBuilder,

			Inline:      inlineRenderer,
			Helm:        helmRenderer,
			Local:       localRenderer,
			Docker:      dockerRenderer,
			DockerLayer: dockerlayers,
			GitHub:      gh,
			Terraform:   tf,
			Web:         webRenderer,
			AmazonEKS:   amazonEKS,
			GoogleGKE:   googleGKE,
			AzureAKS:    azureAKS,
			Status:      status,
		}
	})

}
