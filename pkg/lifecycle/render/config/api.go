package config

import (
	"context"
	"encoding/json"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/viper"
)

// APIResolver resolves config values via API
type APIResolver struct {
	Logger  log.Logger
	Release *api.Release
	UI      cli.Ui
	Viper   *viper.Viper
}

// ResolveConfig will get all the config values specified in the spec, in JSON format
func (r *APIResolver) ResolveConfig(metadata *api.ReleaseMetadata, ctx context.Context) (map[string]interface{}, error) {
	resolvedConfig := make([]map[string]interface{}, 0, 0)

	builder := NewBuilder(
		StaticCtx{},
	)

	for _, configGroup := range r.Release.Spec.Config.V1 {
		resolvedItems := make([]map[string]interface{}, 0, 0)
		for _, configItem := range configGroup.Items {
			resolvedItem, err := r.resolveConfigItem(builder, configItem, ctx)
			if err != nil {

			}

			resolvedItems = append(resolvedItems, resolvedItem)
		}

		resolvedGroup, err := r.resolveConfigGroup(builder, &configGroup, ctx)
		if err != nil {

		}

		resolvedConfig = append(resolvedConfig, resolvedGroup)
	}

	// TODO change the interface to make this a better fit
	fit := make(map[string]interface{})
	fit["config"] = resolvedConfig
	return fit, nil
}

func (r *APIResolver) resolveConfigGroup(builder Builder, configGroup *libyaml.ConfigGroup, ctx context.Context) (map[string]interface{}, error) {
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

func (r *APIResolver) resolveConfigItem(builder Builder, configItem *libyaml.ConfigItem, ctx context.Context) (map[string]interface{}, error) {
	var filters []string
	for _, filter := range configItem.Filters {
		builtFilter, err := builder.String(filter)
		if err != nil {
			r.Logger.Log("msg", err)
			return nil, err
		}
		filters = append(filters, builtFilter)
	}

	b, err := json.Marshal(configItem)
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
