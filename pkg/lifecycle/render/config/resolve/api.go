package resolve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

func NewRenderer(
	logger log.Logger,
	v *viper.Viper,
	builderBuilder *templates.BuilderBuilder,
) *APIConfigRenderer {
	return &APIConfigRenderer{
		Logger:         logger,
		Viper:          v,
		BuilderBuilder: builderBuilder,
	}
}

const MissingRequiredValue = "MISSING_REQUIRED_VALUE"

// APIConfigRenderer resolves config values via API
type APIConfigRenderer struct {
	Logger         log.Logger
	Viper          *viper.Viper
	BuilderBuilder *templates.BuilderBuilder
}

type ValidationError struct {
	Message   string `json:"message"`
	Name      string `json:"name"`
	ErrorCode string `json:"error_code"`
}

func isReadOnly(item *libyaml.ConfigItem) bool {
	if item.ReadOnly {
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

func (r *APIConfigRenderer) shouldOverrideValueWithDefault(item *libyaml.ConfigItem, savedState map[string]interface{}, firstPass bool) bool {
	// resolve config runs before values are saved in interactive mode.
	// this first pass should override any hidden, empty values with
	// non-empty defaults
	if firstPass {
		return item.Hidden && item.Value == "" && item.Default != ""
	}
	// vendor can't override a default with "" in interactive mode
	_, ok := savedState[item.Name]
	return !ok && item.Value == "" && item.Default != ""
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
func (r *APIConfigRenderer) resolveConfigValuesMap(
	meta api.ReleaseMetadata,
	liveValues map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) (map[string]interface{}, error) {
	// make a deep copy of the live values map
	updatedValues, err := deepCopyMap(liveValues)
	if err != nil {
		return nil, errors.Wrap(err, "deep copy live values")
	}

	//recalculate builder with new values
	builder, err := r.BuilderBuilder.FullBuilder(
		meta,
		configGroups,
		updatedValues,
	)
	if err != nil {
		return nil, errors.Wrap(err, "init builder")
	}

	configItemsByName := make(map[string]*libyaml.ConfigItem)
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			configItemsByName[configItem.Name] = configItem
		}
	}

	// Build config values in order & add them to the template builder
	deps := depGraph{
		BuilderBuilder: r.BuilderBuilder,
	}
	err = deps.ParseConfigGroup(configGroups)
	if err != nil {
		return nil, errors.Wrap(err, "parse config groups")
	}
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
		builder, err = r.BuilderBuilder.FullBuilder(
			meta,
			configGroups,
			updatedValues,
		)
		if err != nil {
			return nil, errors.Wrap(err, "re-init builder")
		}
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
	savedState map[string]interface{},
	liveValues map[string]interface{},
	firstPass bool,
) ([]libyaml.ConfigGroup, error) {
	resolvedConfig := make([]libyaml.ConfigGroup, 0)
	configCopy, err := r.deepCopyConfig(release.Spec.Config.V1)
	if err != nil {
		return resolvedConfig, errors.Wrap(err, "deep copy config")
	}

	combinedState, err := deepCopyMap(liveValues)
	if err != nil {
		return resolvedConfig, errors.Wrap(err, "deep copy state")
	}
	for key, val := range savedState {
		if _, ok := combinedState[key]; !ok {
			combinedState[key] = val
		}
	}

	updatedValues, err := r.resolveConfigValuesMap(release.Metadata, combinedState, configCopy)

	if err != nil {
		return resolvedConfig, errors.Wrap(err, "resolve configCopy values map")
	}

	builder, err := r.BuilderBuilder.FullBuilder(release.Metadata, resolvedConfig, updatedValues)
	if err != nil {
		return resolvedConfig, errors.Wrap(err, "initialize tpl builder")
	}

	for _, configGroup := range configCopy {
		resolvedItems := make([]*libyaml.ConfigItem, 0)
		for _, configItem := range configGroup.Items {
			if !isReadOnly(configItem) {
				if val, ok := combinedState[configItem.Name]; ok {
					configItem.Value = fmt.Sprintf("%s", val)
				}
			}

			resolvedItem, err := r.applyConfigItemFieldTemplates(ctx, *builder, configItem, updatedValues)
			if err != nil {
				return resolvedConfig, errors.Wrapf(err, "resolve item %s", configItem.Name)
			}

			if r.shouldOverrideValueWithDefault(configItem, savedState, firstPass) {
				configItem.Value = configItem.Default
			}

			resolvedItems = append(resolvedItems, resolvedItem)
		}

		configGroup.Items = resolvedItems

		resolvedGroup, err := r.applyConfigGroupFieldTemplates(ctx, *builder, configGroup)
		if err != nil {
			return resolvedConfig, errors.Wrapf(err, "resolve group %s", configGroup.Name)
		}

		resolvedConfig = append(resolvedConfig, resolvedGroup)
	}

	return resolvedConfig, nil
}

// ValidateConfig validates a list of resolved config items
func ValidateConfig(
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
	if isRequired(configItem) && !(isReadOnly(configItem) || isHidden(configItem)) {
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

func (r *APIConfigRenderer) applyConfigGroupFieldTemplates(ctx context.Context, builder templates.Builder, configGroup libyaml.ConfigGroup) (libyaml.ConfigGroup, error) {
	// configgroup doesn't have a hidden attribute, so if the config group is hidden, we should
	// set all items as hidden. this is called after applyConfigItemFieldTemplates and will override all hidden
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

func (r *APIConfigRenderer) applyConfigItemFieldTemplates(ctx context.Context, builder templates.Builder, configItem *libyaml.ConfigItem, configValues map[string]interface{}) (*libyaml.ConfigItem, error) {
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

	previousVal, ok := configValues[configItem.Name]

	if ok {
		// only use this for defaults/values that should exist
		if configItem.Value != "" {
			configItem.Value = fmt.Sprintf("%s", previousVal)
		} else {
			configItem.Default = fmt.Sprintf("%s", previousVal)
		}
	}

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
		childItems := make([]*libyaml.ConfigChildItem, 0)
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

func (r *APIConfigRenderer) resolveConfigChildItem(ctx context.Context, builder templates.Builder, configChildItem *libyaml.ConfigChildItem) (*libyaml.ConfigChildItem, error) {
	// TODO
	return configChildItem, nil
}
func (r *APIConfigRenderer) deepCopyConfig(groups []libyaml.ConfigGroup) ([]libyaml.ConfigGroup, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(groups)
	if err != nil {
		return nil, errors.Wrapf(err, "encode group")
	}

	level.Debug(r.Logger).Log("event", "deepCopyConfig.encode", "encoded", buf.String())

	var groupsCopy []libyaml.ConfigGroup
	err = dec.Decode(&groupsCopy)
	if err != nil {
		return nil, errors.Wrapf(err, "decode group")
	}
	return groupsCopy, nil
}
