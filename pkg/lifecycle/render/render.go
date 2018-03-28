package render

import (
	"context"
	"fmt"

	"bytes"
	"html/template"

	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
)

// todo param this or something
const StateFilePath = ".ship/state.json"

// A Renderer takes a resolved spec, collects config values, and renders assets
type Renderer struct {
	Logger         log.Logger
	ConfigResolver *ConfigResolver

	Fs   afero.Afero
	Spec *api.Spec
}

// A Plan is a list of PlanSteps to execute
type Plan []PlanStep

// A PlanStep describes a single unit of work that Ship will do
// to render the application
type PlanStep struct {
	Description string `json:"plan" yaml:"plan" hcl:"plan"`
	Execute     func(ctx context.Context) error
	Err         error
}

// Execute renders the assets and config
func (r *Renderer) Execute(ctx context.Context, step *api.Render) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute", "step.plan", step.SkipPlan)
	var plan Plan

	templateContext, err := r.ConfigResolver.ResolveConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	debug.Log("event", "render.plan")
	for _, asset := range r.Spec.Assets.V1 {
		if asset.Inline != nil {
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, r.inlineStep(asset.Inline, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%v", asset))
		}
	}

	if !step.SkipPlan {
		// print plan
		// confirm plan
	}

	var multiError *multierror.Error

	for _, step := range plan {
		multiError = multierror.Append(multiError, step.Execute(ctx))
	}

	if multiError.ErrorOrNil() != nil {
		return errors.Wrapf(multiError, "execute plan")
	}

	// if not studio:
	//      save state
	// else:
	//      warnStudio
	return nil
}

func (r *Renderer) inlineStep(inline *api.InlineAsset, templateContext map[string]interface{}) PlanStep {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline"))
	description := fmt.Sprintf("Generate inline file %s", inline.Dest)
	return PlanStep{
		Description: description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")
			tpl, err := template.New(description).
				Delims("{{ship ", "}}").
				Funcs(r.funcMap(templateContext)).
				Parse(inline.Contents)
			if err != nil {
				return errors.Wrapf(err, "Parse template for asset at %s", inline.Dest)
			}

			var rendered bytes.Buffer
			err = tpl.Execute(&rendered, templateContext)
			if err != nil {
				return errors.Wrapf(err, "Execute template for asset at %s", inline.Dest)
			}

			basePath := filepath.Dir(inline.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", inline.Dest, "basePath", basePath)
			if err := r.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", inline.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", inline.Dest)
			}

			if err := r.Fs.WriteFile(inline.Dest, rendered.Bytes(), inline.Mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write inline asset to %s", inline.Dest)
			}
			return nil
		},
	}
}
func (r *Renderer) funcMap(templateContext map[string]interface{}) template.FuncMap {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "template"))

	return map[string]interface{}{
		"config": func(name string) interface{} {
			configItemValue, ok := templateContext[name]
			if !ok {
				debug.Log("event", "template.missing", "func", "config", "requested", name, "context", templateContext)
				return ""
			}
			return configItemValue
		},
		"context": func(name string) interface{} {
			switch name {
			case "state_file_path":
				return StateFilePath
			}
			debug.Log("event", "template.missing", "func", "context", "requested", name)
			return ""
		},
	}
}
