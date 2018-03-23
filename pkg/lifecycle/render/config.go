package render

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// ConfigResolver resolves config values via CLI or UI
type ConfigResolver struct {
	Fs     afero.Afero
	Logger log.Logger
	Spec   *api.Spec
	UI     cli.Ui
	Viper  *viper.Viper
}

// ResolveConfig will get all the config values specified in the spec
func (c *ConfigResolver) ResolveConfig(ctx context.Context) (map[string]interface{}, error) {
	debug := level.Debug(log.With(c.Logger, "step.type", "render"))
	debug.Log("event", "config.resolve")

	templateContext := make(map[string]interface{})
	c.Viper.Unmarshal(&templateContext)

	// read runner.spec.config
	for _, configGroup := range c.Spec.Config.V1 {
		c.UI.Info(fmt.Sprintf("Configuring %s", configGroup.Name))
		for _, configItem := range configGroup.Items {
			current, ok := templateContext[configItem.Name]
			if !ok {
				current = ""
			} else {
				current = fmt.Sprintf(" [%s]", current)
			}

			debug.Log("event", "configitem.ask", "group", configGroup.Name, "item", configItem.Name, "type", configItem.Type)
			answer, err := c.UI.Ask(fmt.Sprintf(`Enter a value for option "%s"%s:`, configItem.Name, current))
			if err != nil {
				return nil, errors.Wrapf(err, "Ask value for config option %s", configItem.Name)
			}
			debug.Log("event", "ui.answer", "group", configGroup.Name, "item", configItem.Name, "type", configItem.Type, "answer", answer)
			if answer != "" {
				templateContext[configItem.Name] = answer
			}
		}
	}
	return templateContext, nil

}
