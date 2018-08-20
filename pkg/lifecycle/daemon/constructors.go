package daemon

import (
	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

type OptionalRoutes struct {
	dig.In
	V2Router *NavcycleRoutes `optional:"true"`
	V1Router *V1Routes       `optional:"true"`
}

func NewHeadedDaemon(
	logger log.Logger,
	v *viper.Viper,
	webUIFactory WebUIBuilder,
	routes OptionalRoutes,
) daemontypes.Daemon {
	return &ShipDaemon{
		Logger:         log.With(logger, "struct", "daemon"),
		WebUIFactory:   webUIFactory,
		Viper:          v,
		ExitChan:       make(chan error),
		V1Routes:       routes.V1Router,
		NavcycleRoutes: routes.V2Router,
	}
}

func NewV2Router(
	logger log.Logger,
	stateManager state.Manager,
	messenger lifecycle.Messenger,
	helmIntro lifecycle.HelmIntro,
	helmValues lifecycle.HelmValues,
	kustomizeIntro lifecycle.KustomizeIntro,
	kustomizer lifecycle.Kustomizer,
	configRenderer *resolve.APIConfigRenderer,
	planners planner.Planner,
	patcher patch.Patcher,
	renderer lifecycle.Renderer,
	treeLoader filetree.Loader,
	fs afero.Afero,
) *NavcycleRoutes {
	return &NavcycleRoutes{
		Logger:       logger,
		StateManager: stateManager,
		Planner:      planners,
		Shutdown:     make(chan interface{}),

		Messenger:      messenger,
		HelmIntro:      helmIntro,
		HelmValues:     helmValues,
		KustomizeIntro: kustomizeIntro,
		Kustomizer:     kustomizer,
		ConfigRenderer: configRenderer,
		Patcher:        patcher,
		Renderer:       renderer,
		StepExecutor: func(d *NavcycleRoutes, step api.Step) error {
			return d.execute(step)
		},
		TreeLoader:   treeLoader,
		StepProgress: &daemontypes.ProgressMap{},
		Fs:           fs,
	}
}

func NewV1Router(
	v *viper.Viper,
	renderer *resolve.APIConfigRenderer,
	stateManager state.Manager,
	logger log.Logger,
	ui cli.Ui,
	fs afero.Afero,
	treeLoader filetree.Loader,
	patcher patch.Patcher,
) *V1Routes {
	return &V1Routes{
		Logger:             log.With(logger, "routes", "v1"),
		Fs:                 fs,
		UI:                 ui,
		StateManager:       stateManager,
		Viper:              v,
		TreeLoader:         treeLoader,
		Patcher:            patcher,
		ConfigSaved:        make(chan interface{}, 1),
		MessageConfirmed:   make(chan string, 1),
		TerraformConfirmed: make(chan bool, 1),
		KustomizeSaved:     make(chan interface{}, 1),
		ConfigRenderer:     renderer,
		OpenWebConsole:     tryOpenWebConsole,
	}
}
