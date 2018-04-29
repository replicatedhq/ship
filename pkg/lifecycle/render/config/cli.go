package config

import (
	"context"
	"fmt"
	"reflect"

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
	Logger  log.Logger
	Release *api.Release
	UI      cli.Ui
	Viper   *viper.Viper
}

// ResolveConfig will get all the config values specified in the spec
func (c *CLIResolver) ResolveConfig(metadata *api.ReleaseMetadata, ctx context.Context) (map[string]interface{}, error) {
	debug := level.Debug(log.With(c.Logger, "step.type", "render"))
	debug.Log("event", "config.resolve")

	templateContext := make(map[string]interface{})
	c.Viper.Unmarshal(&templateContext)

	// read runner.spec.config
	for _, configGroup := range c.Release.Spec.Config.V1 {
		c.UI.Info(configGroup.Name)
		for _, configItem := range configGroup.Items {
			current := c.resolveCurrentValue(templateContext, configItem)

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

	if metadata != nil {
		c.resolveMetadata(metadata, templateContext)
	}

	return templateContext, nil
}

func (c *CLIResolver) resolveCurrentValue(templateContext map[string]interface{}, configItem *libyaml.ConfigItem) interface{} {
	debug := log.With(level.Debug(c.Logger), "func", "resolve-current", "config-item", configItem.Name)
	// check ctx first
	current, ok := templateContext[configItem.Name]
	if ok {
		debug.Log("event", "templateContext.ok")
		return current
	}

	//then check viper
	current = c.Viper.Get(configItem.Name)
	if current != "" && current != nil {
		debug.Log("event", "viper.found", "value", current)
		return current
	}

	//then use default viper
	debug.Log("event", "use.default", "empty", configItem.Default == "")
	return configItem.Default
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

func (c *CLIResolver) resolveMetadata(metadata *api.ReleaseMetadata, templateContext map[string]interface{}) {
	t := reflect.TypeOf(*metadata)
	v := reflect.ValueOf(*metadata)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("meta")
		if tag == "" || tag == "-" {
			continue
		}
		if val, ok := templateContext[tag]; ok {
			if v, ok := val.(string); ok && v != "" {
				continue // should we override or not?  it makes no difference at the moment.
			}
		}
		templateContext[tag] = v.Field(i).Interface()
	}
}
