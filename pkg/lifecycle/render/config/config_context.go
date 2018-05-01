package config

import (
	"fmt"
	"os"
	"text/template"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// NewConfigContext will return a new config context, initialized with the app config.
// Once we have state (for upgrades) it should be a parameter here.
func NewConfigContext(configGroups []libyaml.ConfigGroup, pendingValues []ItemValue) (*ConfigCtx, error) {
	// Get a static context to render static template functions

	builder := NewBuilder(
		StaticCtx{},
	)

	configCtx := &ConfigCtx{
		ItemValues: make(map[string]string),
		Logger:     log.NewLogfmtLogger(os.Stderr),
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

			// This is super raw, unefficient and needs some ♡ before it should be 🚢'ed
			for _, pendingValue := range pendingValues {
				if pendingValue.Name == configItem.Name {
					if pendingValue.Value != built {
						built = pendingValue.Value
					}
				}
			}

			configCtx.ItemValues[configItem.Name] = built
		}
	}

	return configCtx, nil
}

// ConfigCtx is the context for builder functions before the application has started.
type ConfigCtx struct {
	ItemValues map[string]string
	Logger     log.Logger
}

// FuncMap represents the available functions in the ConfigCtx.
func (ctx ConfigCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"ConfigOption":          ctx.configOption,
		"ConfigOptionIndex":     ctx.configOptionIndex,
		"ConfigOptionData":      ctx.configOptionData,
		"ConfigOptionEquals":    ctx.configOptionEquals,
		"ConfigOptionNotEquals": ctx.configOptionNotEquals,
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
		return val, nil
	}

	err := fmt.Errorf("unable to find config item named %q", itemName)
	level.Error(ctx.Logger).Log("msg", "unable to find config item", "err", err)
	return "", err
}
