package config

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedhq/libyaml"
	"github.com/spf13/viper"
)

// CLIResolver resolves config values via CLI
type CLIResolver struct {
	Logger log.Logger
	Spec   *api.Spec
	UI     cli.Ui
	Viper  *viper.Viper
}

// ResolveConfig will get all the config values specified in the spec
func (c *CLIResolver) ResolveConfig(ctx context.Context) (map[string]interface{}, error) {
	debug := level.Debug(log.With(c.Logger, "step.type", "render"))
	debug.Log("event", "config.resolve")

	templateContext := make(map[string]interface{})
	c.Viper.Unmarshal(&templateContext)

	// read runner.spec.config
	for _, configGroup := range c.Spec.Config.V1 {
		c.UI.Info(configGroup.Name)
		for _, configItem := range configGroup.Items {
			current := resolveCurrentValue(templateContext, configItem)

			for {
				debug.Log("event", "configitem.ask", "group", configGroup.Name, "item", configItem.Name, "type", configItem.Type)
				answer, err := c.UI.Ask(fmt.Sprintf(`Enter a value for option "%s"%s:`, configItem.Name, formatCurrent(configItem, current)))
				if err != nil {
					return nil, errors.Wrapf(err, "Ask value for config option %s", configItem.Name)
				}
				debug.Log("event", "ui.answer", "group", configGroup.Name, "item", configItem.Name, "type", configItem.Type, "answer", answer)

				if answer == "" && current == "" && configItem.Required {
					c.UI.Warn(fmt.Sprintf(`Option "%s" is required`, configItem.Name))
					continue
				}

				if answer != "" {
					templateContext[configItem.Name] = answer
				} else {
					templateContext[configItem.Name] = current
				}
				break
			}
		}
	}
	return templateContext, nil

}

func resolveCurrentValue(templateContext map[string]interface{}, configItem *libyaml.ConfigItem) interface{} {
	current, ok := templateContext[configItem.Name]
	if !ok {
		if configItem.Default != "" {
			return configItem.Default
		}
		return ""
	}

	return current
}

func formatCurrent(configItem *libyaml.ConfigItem, current interface{}) string {
	if current == nil || current == "" {
		return ""
	}

	if configItem.Type == "password" {
		return fmt.Sprintf(" [xxxx%3s]", current)
	}

	return fmt.Sprintf(" [%s]", current)
}
