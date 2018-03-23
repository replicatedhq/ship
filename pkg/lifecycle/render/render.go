package render

import (
	"context"
	"fmt"

	"bytes"
	"html/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// A Renderer takes a resolved spec, collects config values, and renders assets
type Renderer struct {
	ConfigResolver *ConfigResolver

	Step   *api.Render
	Fs     afero.Afero
	Logger log.Logger
	Spec   *api.Spec
	UI     cli.Ui
	Viper  *viper.Viper
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
func (r *Renderer) Execute(ctx context.Context) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render"))
	debug.Log("event", "step.execute", "step.plan", r.Step.SkipPlan)
	var plan Plan

	templateContext, err := r.ConfigResolver.ResolveConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "resolve config")
	}

	// gather config values
	// store to temp state
	// confirm? (obfuscating passwords)

	debug.Log("event", "render.plan")
	for _, asset := range r.Spec.Assets.V1 {
		if asset.Inline != nil {
			debug.Log("event", "asset.resolve", "asset.type", "inline")
			plan = append(plan, r.inlineStep(asset.Inline, templateContext))
		} else {
			debug.Log("event", "asset.resolve.fail", "asset", fmt.Sprintf("%v", asset))
		}
	}

	if !r.Step.SkipPlan {
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

func (r *Renderer) String() string {
	return fmt.Sprintf("Render{SkipPlan=%v}", r.Step.SkipPlan)
}

func (r *Renderer) inlineStep(inline *api.InlineAsset, templateContext map[string]interface{}) PlanStep {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "inline"))
	description := fmt.Sprintf("Generate inline file %s", inline.Dest)
	return PlanStep{
		Description: description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")
			tpl, err := template.New(description).Parse(inline.Contents)
			if err != nil {
				return errors.Wrapf(err, "Parse template for asset at %s", inline.Dest)
			}
			var rendered bytes.Buffer
			err = tpl.Execute(&rendered, templateContext)
			if err != nil {
				return errors.Wrapf(err, "Execute template for asset at %s", inline.Dest)
			}

			if err := r.Fs.WriteFile(inline.Dest, rendered.Bytes(), inline.Mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write inline asset to %s", inline.Dest)
			}
			return nil
		},
	}
}
