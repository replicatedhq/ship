package daemon

import (
	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/patch"
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
	v *viper.Viper,
	renderer *resolve.APIConfigRenderer,
	stateManager state.Manager,
	logger log.Logger,
	ui cli.Ui,
	fs afero.Afero,
	webUIFactory WebUIBuilder,
	treeLoader filetree.Loader,
	patcher patch.Patcher,
) *ShipDaemon {
	return &ShipDaemon{
		Logger:             log.With(logger, "struct", "daemon"),
		Fs:                 fs,
		UI:                 ui,
		StateManager:       stateManager,
		Viper:              v,
		WebUIFactory:       webUIFactory,
		TreeLoader:         treeLoader,
		Patcher:            patcher,
		ConfigSaved:        make(chan interface{}, 1),
		MessageConfirmed:   make(chan string, 1),
		TerraformConfirmed: make(chan bool, 1),
		KustomizeSaved:     make(chan interface{}, 1),
		ConfigRenderer:     renderer,
		OpenWebConsole:     tryOpenWebConsole,
		exitChan:           make(chan error),
	}
}
