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
	V2Router *V2Routes `optional:"true"`
	V1Router *V1Routes `optional:"true"`
}

func NewHeadedDaemon(
	logger log.Logger,
	v *viper.Viper,
	webUIFactory WebUIBuilder,
	routes OptionalRoutes,
) daemontypes.Daemon {
	return &ShipDaemon{
		Logger:       log.With(logger, "struct", "daemon"),
		WebUIFactory: webUIFactory,
		Viper:        v,
		ExitChan:     make(chan error),
		V1Routes:     routes.V1Router,
		V2Routes:     routes.V2Router,
	}
}

func NewV2Router(
	logger log.Logger,
	stateManager state.Manager,
	messenger lifecycle.Messenger,
	helmIntro lifecycle.HelmIntro,
	planners planner.Planner,
	renderer lifecycle.Renderer,
) *V2Routes {
	return &V2Routes{
		Logger:       logger,
		StateManager: stateManager,
		Planner:      planners,

		Messenger: messenger,
		HelmIntro: helmIntro,
		Renderer:  renderer,
		StepExecutor: func(d *V2Routes, step api.Step) error {
			return d.execute(step)
		},

		StepProgress: make(map[string]daemontypes.Progress),
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
