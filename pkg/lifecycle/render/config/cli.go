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

	"github.com/replicatedcom/ship/pkg/templates"
)

var _ Resolver = &CLIResolver{}

// CLIResolver resolves config values via CLI
type CLIResolver struct {
	Logger log.Logger
	UI     cli.Ui
	Viper  *viper.Viper
}

func (r *CLIResolver) WithDaemon(_ Daemon) Resolver {
	return r
}

// ResolveConfig will get all the config values specified in the spec
func (c *CLIResolver) ResolveConfig(
	ctx context.Context,
	release *api.Release,
	templateContext map[string]interface{},
) (map[string]interface{}, error) {
	debug := level.Debug(log.With(c.Logger, "step.type", "render"))
	debug.Log("event", "config.resolve")

	c.Viper.Unmarshal(&templateContext)

	configCtx, err := NewConfigContext(
		c.Viper, c.Logger,
		release.Spec.Config.V1, templateContext,
	)
	if err != nil {
		return nil, err
	}

	builder := templates.NewBuilder(
		templates.NewStaticContext(),
		configCtx,
	)

	// read runner.spec.config
	for _, configGroup := range release.Spec.Config.V1 {
		c.UI.Info(configGroup.Name)
		for _, configItem := range configGroup.Items {
			current := c.resolveCurrentValue(templateContext, builder, configItem)

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

	c.resolveMetadata(&release.Metadata, templateContext)

	return templateContext, nil
}

func (c *CLIResolver) resolveCurrentValue(templateContext map[string]interface{}, builder templates.Builder, configItem *libyaml.ConfigItem) interface{} {
	debug := log.With(level.Debug(c.Logger), "func", "resolve-current", "config-item", configItem.Name)

	// check env first
	if c.Viper.GetString(configItem.Name) != "" {
		debug.Log("event", "env.ok")
		return c.Viper.GetString(configItem.Name)
	}

	// check ctx first
	current, ok := templateContext[configItem.Name]
	if ok {
		debug.Log("event", "templateContext.ok")
		built, err := builder.String(current.(string))
		if err != nil {
			return errors.Wrapf(err, "builder.string %q", current)
		}
		return built
	}

	//then check viper
	current = c.Viper.Get(configItem.Name)
	if current != "" && current != nil {
		debug.Log("event", "viper.found", "value", current)

		built, err := builder.String(current.(string))
		if err != nil {
			return errors.Wrapf(err, "builder.string %q", current)
		}
		return built
	}

	//then use default viper
	debug.Log("event", "use.default", "empty", configItem.Default == "")
	built, err := builder.String(configItem.Default)
	if err != nil {
		return errors.Wrapf(err, "builder.string %q", configItem.Default)
	}
	return built
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
