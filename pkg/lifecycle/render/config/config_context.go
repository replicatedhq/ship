package config

import (
	"fmt"
	"text/template"

	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"
)

// NewConfigContext will return a new config context, initialized with the app config.
// Once we have state (for upgrades) it should be a parameter here.
func NewConfigContext(viper *viper.Viper, logger log.Logger, configGroups []libyaml.ConfigGroup, templateContext map[string]interface{}) (*ConfigCtx, error) {
	// Get a static context to render static template functions

	builder := NewBuilder(
		StaticCtx{},
	)

	configCtx := &ConfigCtx{
		ItemValues:       templateContext,
		ItemDependencies: depGraph{},
		Logger:           logger,
		Viper:            viper,
	}

	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// if the pending value is different from the built, then use the pending every time
			// We have to ignore errors here because we only have the static context loaded
			// for rendering. some items have templates that need the config context,
			// so we can ignore these.
			builtDefault, _ := builder.String(configItem.Default)
			builtValue, _ := builder.String(configItem.Value)

			var built string
			if builtValue != "" {
				built = builtValue
			} else {
				built = builtDefault
			}

			if v, ok := templateContext[configItem.Name]; ok {
				built = fmt.Sprintf("%v", v)
			}

			configCtx.ItemValues[configItem.Name] = built

			// add this to the dependency graph
			depBuilder := NewBuilder()
			depBuilder.Functs = configCtx.ItemDependencies.FuncMap(configItem.Name)

			depBuilder.String(configItem.Default)
			depBuilder.String(configItem.Value)
		}
	}

	fmt.Println(configCtx.ItemDependencies.PrintData())

	return configCtx, nil
}

// ConfigCtx is the context for builder functions before the application has started.
type ConfigCtx struct {
	ItemValues       map[string]interface{}
	ItemDependencies depGraph
	Logger           log.Logger
	Viper            *viper.Viper
}

// FuncMap represents the available functions in the ConfigCtx.
func (ctx ConfigCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ConfigOption":          ctx.configOption,
		"ConfigOptionIndex":     ctx.configOptionIndex,
		"ConfigOptionData":      ctx.configOptionData,
		"ConfigOptionEquals":    ctx.configOptionEquals,
		"ConfigOptionNotEquals": ctx.configOptionNotEquals,

		// this should probably go somewhere else eventually
		// Install should have all the details about this ship installation,
		// including customer Id, customer name release notes, version, etc.
		"Installation": ctx.Install,
		// old, remove
		"context": ctx.Install,
	}
}

func (ctx ConfigCtx) Install(name string) string {
	switch name {
	case "state_file_path":
		return state.Path
	case "customer_id":
		return ctx.Viper.GetString("customer-id")
	default:
		level.Warn(ctx.Logger).Log("event", "ConfigCtx.context.unsuppported", "name", name)
		return ""
	}
}

func (ctx ConfigCtx) configOption(name string) string {
	v, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return ""
	}
	return v
}

func (ctx ConfigCtx) configOptionIndex(name string) string {
	return ""
}

func (ctx ConfigCtx) configOptionData(name string) string {
	return ""
}

func (ctx ConfigCtx) configOptionEquals(name string, value string) bool {
	val, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return false
	}

	return value == val
}

func (ctx ConfigCtx) configOptionNotEquals(name string, value string) bool {
	val, err := ctx.getConfigOptionValue(name)
	if err != nil {
		return false
	}

	return value != val
}

func (ctx ConfigCtx) getConfigOptionValue(itemName string) (string, error) {
	if val, ok := ctx.ItemValues[itemName]; ok {
		return fmt.Sprintf("%v", val), nil
	}

	err := fmt.Errorf("unable to find config item named %q", itemName)
	level.Error(ctx.Logger).Log("msg", "unable to find config item", "err", err)
	return "", err
}
