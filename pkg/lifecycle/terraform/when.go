package terraform

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

func (t *ForkTerraformer) evaluateWhen(when string, release api.Release) bool {
	debug := level.Debug(log.With(t.Logger, "struct", "ForkTerraformer", "method", "execute"))

	return evaluateWhen(when, release, debug, t.Viper, t.StateManager)
}

func (t *DaemonlessTerraformer) evaluateWhen(when string, release api.Release) bool {
	debug := level.Debug(log.With(t.Logger, "struct", "DaemonlessTerraformer", "method", "execute"))

	return evaluateWhen(when, release, debug, t.Viper, t.StateManager)
}

func evaluateWhen(when string, release api.Release, logger log.Logger, viper *viper.Viper, manager state.Manager) bool {
	if manager == nil {
		return true
	}

	builderBuilder := templates.BuilderBuilder{Logger: logger, Viper: viper, Manager: manager}

	configState, err := manager.CachedState()
	if err != nil {
		_ = logger.Log("terraform.when.loadState", err.Error())
	}

	currentConfig, err := configState.CurrentConfig()
	if err != nil {
		_ = logger.Log("terraform.when.getConfig", err.Error())
	}

	fullBuilder, err := builderBuilder.FullBuilder(release.Metadata, release.Spec.Config.V1, currentConfig)
	if err != nil {
		_ = logger.Log("terraform.when.buildBuilder", err.Error())
	}

	build, err := fullBuilder.Bool(when, true)

	if err != nil {
		_ = logger.Log("terraform.when.error", err.Error())
		// the terraform step should be run if the template function returns an error
		return true
	}

	return build
}
