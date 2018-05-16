package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/replicatedcom/ship/pkg/api"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"
)

const MissingRequiredValue = "MISSING_REQUIRED_VALUE"

// APIConfigRenderer resolves config values via API
type APIConfigRenderer struct {
	Logger log.Logger
	Viper  *viper.Viper
}

type ValidationError struct {
	Message   string `json:"message"`
	Name      string `json:"name"`
	ErrorCode string `json:"error_code"`
}

func isReadOnly(item *libyaml.ConfigItem) bool {
	if item.ReadOnly || item.Hidden {
		return true
	}

	// "" is an editable type because the default type is "text"
	var EditableItemTypes = map[string]struct{}{
		"":            {},
		"bool":        {},
		"file":        {},
		"password":    {},
		"select":      {},
		"select_many": {},
		"select_one":  {},
		"text":        {},
		"textarea":    {},
	}

	_, editable := EditableItemTypes[item.Type]
	return !editable
}

func isRequired(item *libyaml.ConfigItem) bool {
	return item.Required
}

func isEmpty(item *libyaml.ConfigItem) bool {
	return item.Value == "" && item.Default == ""
}

func isHidden(item *libyaml.ConfigItem) bool {
	return item.Hidden
}

func deepCopyMap(original map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(original)
	if err != nil {
		return nil, err
	}
	var updatedValues map[string]interface{}
	err = dec.Decode(&updatedValues)
	if err != nil {
		return nil, err
	}
	return updatedValues, nil
}

// given a set of input values ('liveValues') and the config ('configGroups') returns a map of configItem names to values, with all config option template functions resolved
func resolveConfigValuesMap(liveValues map[string]interface{}, configGroups []libyaml.ConfigGroup, logger log.Logger, viper *viper.Viper) (map[string]interface{}, error) {
	// make a deep copy of the live values map
	updatedValues, err := deepCopyMap(liveValues)
	if err != nil {
		return nil, err
	}

	staticCtx, err := NewStaticContext()
	if err != nil {
		return nil, err
	}

	configCtx, err := NewConfigContext(
		viper, logger,
		configGroups,
		updatedValues)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder(
		staticCtx,
		configCtx,
	)

	configItemsByName := make(map[string]*libyaml.ConfigItem)
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			configItemsByName[configItem.Name] = configItem
		}
	}

	// Build config values in order & add them to the template builder
	var deps depGraph
	deps.ParseConfigGroup(configGroups)
	var headNodes []string

	headNodes, err = deps.GetHeadNodes()

	for (len(headNodes) > 0) && (err == nil) {
		for _, node := range headNodes {
			deps.ResolveDep(node)

			configItem := configItemsByName[node]

			if !isReadOnly(configItem) {
				// if item is editable and the live state is valid, skip the rest of this
				val, ok := updatedValues[node]
				if ok && val != "" {
					continue
				}
			}

			// build "default" and "value"
			builtDefault, _ := builder.String(configItem.Default)
			builtValue, _ := builder.String(configItem.Value)

			if builtValue != "" {
				updatedValues[node] = builtValue
			} else {
				updatedValues[node] = builtDefault
			}
		}

		//recalculate builder with new values
		newConfigCtx, err := NewConfigContext(
			viper, logger,
			configGroups,
			updatedValues)
		if err != nil {
			return nil, err
		}

		builder = NewBuilder(
			staticCtx,
			newConfigCtx,
		)

		headNodes, err = deps.GetHeadNodes()
	}
	if err != nil {
		//dependencies could not be resolved for some reason
		//return the empty config
		//TODO: Better error messaging
		return updatedValues, err
	}

	return updatedValues, nil
}

// ResolveConfig will get all the config values specified in the spec, in JSON format
func (r *APIConfigRenderer) ResolveConfig(
	ctx context.Context,
	release *api.Release,
	liveValues map[string]interface{},
) ([]libyaml.ConfigGroup, error) {
	resolvedConfig := make([]libyaml.ConfigGroup, 0, 0)

	updatedValues, err := resolveConfigValuesMap(liveValues, release.Spec.Config.V1, r.Logger, r.Viper)
	if err != nil {
		return resolvedConfig, err
	}

	builder, err := r.newBuilder(ctx, release.Spec.Config.V1, updatedValues)
	if err != nil {
		return resolvedConfig, err
	}

	for _, configGroup := range release.Spec.Config.V1 {
		resolvedItems := make([]*libyaml.ConfigItem, 0, 0)
		for _, configItem := range configGroup.Items {
			if !isReadOnly(configItem) {
				if val, ok := liveValues[configItem.Name]; ok {
					newval := fmt.Sprintf("%v", val)
					if newval != "" {
						configItem.Value = newval
					}
				}
			}

			resolvedItem, err := r.resolveConfigItem(ctx, *builder, configItem)
			if err != nil {
				return resolvedConfig, err
			}

			resolvedItems = append(resolvedItems, resolvedItem)
		}

		configGroup.Items = resolvedItems

		resolvedGroup, err := r.resolveConfigGroup(ctx, *builder, configGroup)
		if err != nil {
			return resolvedConfig, err
		}

		resolvedConfig = append(resolvedConfig, resolvedGroup)
	}

	return resolvedConfig, nil
}

func validateConfig(
	resolvedConfig []libyaml.ConfigGroup,
) []*ValidationError {
	var validationErrs []*ValidationError
	for _, configGroup := range resolvedConfig {
		// hidden is set if when resolves to false

		if hidden := configGroupIsHidden(configGroup); hidden {
			continue
		}

		for _, configItem := range configGroup.Items {

			if invalidItem := validateConfigItem(configItem); invalidItem != nil {
				validationErrs = append(validationErrs, invalidItem)
			}
		}
	}
	return validationErrs
}

func configGroupIsHidden(
	configGroup libyaml.ConfigGroup,
) bool {
	// if all the items in the config group are hidden,
	// we know when is set. thus config group is hidden
	for _, configItem := range configGroup.Items {
		if !isHidden(configItem) {
			return false
		}
	}
	return true
}

func validateConfigItem(
	configItem *libyaml.ConfigItem,
) *ValidationError {
	var validationErr *ValidationError
	if isRequired(configItem) && !isReadOnly(configItem) {
		if isEmpty(configItem) {
			validationErr = &ValidationError{
				Message:   fmt.Sprintf("Config item %s is required", configItem.Name),
				Name:      configItem.Name,
				ErrorCode: MissingRequiredValue,
			}
		}
	}
	return validationErr
}

func (r *APIConfigRenderer) newBuilder(
	ctx context.Context,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) (*Builder, error) {
	staticCtx, err := NewStaticContext()
	if err != nil {
		return nil, err
	}

	newConfigCtx, err := NewConfigContext(
		r.Viper, r.Logger,
		configGroups, templateContext)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder(
		staticCtx,
		newConfigCtx,
	)
	return &builder, nil
}

func (r *APIConfigRenderer) resolveConfigGroup(ctx context.Context, builder Builder, configGroup libyaml.ConfigGroup) (libyaml.ConfigGroup, error) {
	// configgroup doesn't have a hidden attribute, so if the config group is hidden, we should
	// set all items as hidden. this is called after resolveConfigItem and will override all hidden
	// values in items if when is set
	builtWhen, err := builder.String(configGroup.When)
	if err != nil {
		level.Error(r.Logger).Log("msg", "unable to build 'when' on configgroup", "group_name", configGroup.Name, "err", err)
		return libyaml.ConfigGroup{}, err
	}
	configGroup.When = builtWhen

	if builtWhen != "" {
		builtWhenBool, err := builder.Bool(builtWhen, true)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build 'when' bool", "err", err)
			return libyaml.ConfigGroup{}, err
		}

		for _, configItem := range configGroup.Items {
			// if the config group is not hidden, don't override the value in the item
			if !builtWhenBool {
				configItem.Hidden = true
			}
		}
	}

	return configGroup, nil
}

func (r *APIConfigRenderer) resolveConfigItem(ctx context.Context, builder Builder, configItem *libyaml.ConfigItem) (*libyaml.ConfigItem, error) {
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
		builtWhenBool, err := builder.Bool(configItem.When, true)
		if err != nil {
			level.Error(r.Logger).Log("msg", "unable to build 'when'", "err", err)
			return nil, err
		}

		// don't override the hidden value if the when value is false
		if !builtWhenBool {
			configItem.Hidden = true
		}
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

func (r *APIConfigRenderer) resolveConfigChildItem(ctx context.Context, builder Builder, configChildItem *libyaml.ConfigChildItem) (*libyaml.ConfigChildItem, error) {
	// TODO
	return configChildItem, nil
}
