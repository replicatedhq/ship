package daemon

import (
	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func NewHeadlessDaemon(
	v *viper.Viper,
	logger log.Logger,
	renderer *resolve.APIConfigRenderer,
	stateManager *state.Manager,
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
	stateManager *state.Manager,
	logger log.Logger,
	ui cli.Ui,
	fs afero.Afero,
) *ShipDaemon {
	return &ShipDaemon{
		Logger:             logger,
		Fs:                 fs,
		UI:                 ui,
		StateManager:       stateManager,
		Viper:              v,
		ConfigSaved:        make(chan interface{}),
		MessageConfirmed:   make(chan string, 1),
		TerraformConfirmed: make(chan bool, 1),
		ConfigRenderer:     renderer,
		errChan:            make(chan error, 1),
	}

}
