package daemon

import (
	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func NewHeadlessDaemon(
	v *viper.Viper,
	logger log.Logger,
	renderer *resolve.APIConfigRenderer,
	stateManager state.Manager,
) *HeadlessDaemon {
	return &HeadlessDaemon{
		StateManager:   stateManager,
		Logger:         logger,
		UI:             ui.FromViper(v),
		ConfigRenderer: renderer,
	}
}

func NewHeadedDaemon(
	logger log.Logger,
	v *viper.Viper,
	webUIFactory WebUIBuilder,
	v1Router *V1Routes,
	v2Router *V2Routes,
) *ShipDaemon {
	return &ShipDaemon{
		Logger:       log.With(logger, "struct", "daemon"),
		WebUIFactory: webUIFactory,
		Viper:        v,
		exitChan:     make(chan error),
		V1Routes:     v1Router,
		V2Routes:     v2Router,
	}

}

func NewV2Router(
	logger log.Logger,
) *V2Routes {
	return &V2Routes{
		Logger: logger,
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
) *V1Routes {
	return &V1Routes{
		Logger:             log.With(logger, "routes", "v1"),
		Fs:                 fs,
		UI:                 ui,
		StateManager:       stateManager,
		Viper:              v,
		TreeLoader:         treeLoader,
		ConfigSaved:        make(chan interface{}, 1),
		MessageConfirmed:   make(chan string, 1),
		TerraformConfirmed: make(chan bool, 1),
		KustomizeSaved:     make(chan interface{}, 1),
		ConfigRenderer:     renderer,
		OpenWebConsole:     tryOpenWebConsole,
	}

}
