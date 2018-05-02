package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/viper"
)

// APIResolver resolves config values via API
type APIResolver struct {
	Logger  log.Logger
	Release *api.Release
	Viper   *viper.Viper
}

// ResolveConfig will get all the config values specified in the spec, in JSON format
func (r *APIResolver) ResolveConfig(ctx context.Context, metadata *api.ReleaseMetadata, templateContext map[string]interface{}) (map[string]interface{}, error) {

	resolvedConfig := make([]map[string]interface{}, 0, 0)

	staticCtx, err := NewStaticContext()
	if err != nil {
		return nil, err
	}

	configCtx, err := NewConfigContext(r.Release.Spec.Config.V1, templateContext)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder(
		staticCtx,
		configCtx,
	)

	unresolvedConfigItems := make([]*libyaml.ConfigItem, 0, 0)
	for _, configGroup := range r.Release.Spec.Config.V1 {
		for _, configItem := range configGroup.Items {
			unresolvedConfigItems = append(unresolvedConfigItems, configItem)
		}
	}

	for _, configGroup := range r.Release.Spec.Config.V1 {
		resolvedItems := make([]*libyaml.ConfigItem, 0, 0)
		for _, configItem := range configGroup.Items {
			for k, v := range templateContext {
				if k == configItem.Name {
					// (implementation logic copied from replicated 1):
					// this limiation ensures that any config item with a
					// "default" cannot be ""
					if configItem.Default != "" && configItem.Value == "" {
						continue
					}
					configItem.Value = fmt.Sprintf("%v", v)
				}
			}

			resolvedItem, err := r.resolveConfigItem(ctx, builder, configItem)
			if err != nil {
				return nil, err
			}

			resolvedItems = append(resolvedItems, resolvedItem)
		}

		configGroup.Items = resolvedItems

		resolvedGroup, err := r.resolveConfigGroup(ctx, builder, &configGroup)
		if err != nil {
			return nil, err
		}

		resolvedConfig = append(resolvedConfig, resolvedGroup)
	}

	// TODO change the interface to make this a better fit
	fit := make(map[string]interface{})
	fit["config"] = resolvedConfig
	return fit, nil
}

func (r *APIResolver) resolveConfigGroup(ctx context.Context, builder Builder, configGroup *libyaml.ConfigGroup) (map[string]interface{}, error) {
	// configgroup doesn't have a hidden attribute, so if the config group is hidden, we should
	// set all items as hidden
	builtWhen, err := builder.String(configGroup.When)
	if err != nil {
		level.Error(r.Logger).Log("msg", "unable to build 'when' on configgroup", "group_name", configGroup.Name, "err", err)
		return nil, err
	}

	if builtWhen != "" {
		builtWhenBool, err := builder.Bool(builtWhen, true)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build 'when' bool", "err", err)
			return nil, err
		}

		for _, configItem := range configGroup.Items {
			configItem.Hidden = !builtWhenBool
		}
	}

	b, err := json.Marshal(configGroup)
	if err != nil {
		r.Logger.Log("msg", err)
		return nil, err
	}

	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		r.Logger.Log("msg", err)
	}

	return m, nil
}

func (r *APIResolver) resolveConfigItem(ctx context.Context, builder Builder, configItem *libyaml.ConfigItem) (*libyaml.ConfigItem, error) {
	// filters
	var filters []string
	for _, filter := range configItem.Filters {
		builtFilter, err := builder.String(filter)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build filter", "err", err)
			return nil, err
		}
		filters = append(filters, builtFilter)
	}

	// type should default to "text"
	if configItem.Type == "" {
		configItem.Type = "text"
	}

	// build "default"
	builtDefault, err := builder.String(configItem.Default)
	if err != nil {
		level.Error(r.Logger).Log("msg", "unable to build 'default'", "err", err)
		return nil, err
	}
	configItem.Default = builtDefault

	// build "value"
	builtValue, err := builder.String(configItem.Value)
	if err != nil {
		level.Error(r.Logger).Log("msg", "unable to build 'value'", "err", err)
		return nil, err
	}
	configItem.Value = builtValue

	// build "when" (dropping support for the when: a=b style here from replicated v1)
	builtWhen, err := builder.String(configItem.When)
	if err != nil {
		level.Error(r.Logger).Log("msg", "unable to build `when'", "err", err)
		return nil, err
	}
	configItem.When = builtWhen

	// build "runonsave"
	if configItem.TestProc != nil {
		builtRunOnSave, err := builder.Bool(configItem.TestProc.RunOnSave, false)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build 'runonsave'", "err", err)
			return nil, err
		}
		configItem.TestProc.RunOnSave = strconv.FormatBool(builtRunOnSave)
	}

	// build "hidden" from "when" if it's present
	if configItem.When != "" {
		builtWhen, err := builder.Bool(configItem.When, true)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build 'when'", "err", err)
			return nil, err
		}

		configItem.Hidden = !builtWhen
	}

	// build subitems
	if configItem.Items != nil {
		childItems := make([]*libyaml.ConfigChildItem, 0, 0)
		for _, item := range configItem.Items {
			builtChildItem, err := r.resolveConfigChildItem(ctx, builder, item)
			if err != nil {
				return nil, err
			}

			childItems = append(childItems, builtChildItem)
		}

		configItem.Items = childItems
	}

	return configItem, nil
}

func (r *APIResolver) resolveConfigChildItem(ctx context.Context, builder Builder, configChildItem *libyaml.ConfigChildItem) (*libyaml.ConfigChildItem, error) {
	// TODO
	return configChildItem, nil
}
