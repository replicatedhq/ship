package lifecycle

import (
	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// A Runner runs a lifecycle using the passed Spec
type Runner struct {
	CustomerID     string
	InstallationID string
	GraphQLClient  *specs.GraphQLClient
	UI             cli.Ui
	Logger         log.Logger
	Spec           *api.Spec
	Fs             afero.Afero
	Viper          *viper.Viper
}

// Run runs a lifecycle using the passed Spec
func (r *Runner) Run(ctx context.Context) error {
	level.Debug(r.Logger).Log("event", "lifecycle.execute")

	// this needs to be pulled up more, but this is enough for now
	executor := &stepExecutor{
		Logger: r.Logger,
		renderer: &render.Renderer{
			Fs:     r.Fs,
			Logger: r.Logger,
			Spec:   r.Spec,
			ConfigResolver: &render.ConfigResolver{
				Fs:     r.Fs,
				Logger: r.Logger,
				Spec:   r.Spec,
				UI:     r.UI,
				Viper:  r.Viper,
			},
		},
		messenger: &messenger{
			Logger: r.Logger,
			UI:     r.UI,
			Viper:  r.Viper,
		},
	}

	for idx, step := range r.Spec.Lifecycle.V1 {
		level.Debug(r.Logger).Log("event", "step.execute", "index", idx, "step", fmt.Sprintf("%v", step))
		if err := executor.Execute(ctx, &step); err != nil {
			level.Error(r.Logger).Log("event", "step.execute.fail", "index", idx, "step", fmt.Sprintf("%v", step))
			return errors.Wrapf(err, "execute lifecycle step %d: %s", idx, fmt.Sprintf("%v", step))
		}
	}

	return nil
}
