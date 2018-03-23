package lifecycle

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
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

	for idx, step := range r.Spec.Lifecycle.V1 {
		executor := &stepExecutor{&step}
		level.Debug(r.Logger).Log("event", "step.execute", "index", idx, "step", executor.String())
		if err := executor.Execute(ctx, r); err != nil {
			level.Error(r.Logger).Log("event", "step.execute.fail", "index", idx, "step", executor.String())
			return errors.Wrapf(err, "execute lifecycle step %d: %s", idx, executor.String())
		}
	}

	return nil
}
