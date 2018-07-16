package terraform

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"

	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/replicatedhq/ship/pkg/version"
	"github.com/spf13/viper"
)

type Terraformer interface {
	Execute(ctx context.Context, release api.Release, step api.Terraform) error
	WithDaemon(d daemon.Daemon) Terraformer
}

type ForkTerraformer struct {
	Logger        log.Logger
	Daemon        daemon.Daemon
	PlanConfirmer tfplan.PlanConfirmer
	Terraform     func() *exec.Cmd
	Viper         *viper.Viper
}

func NewTerraformer(
	logger log.Logger,
	daemon daemon.Daemon,
	planner tfplan.PlanConfirmer,
	viper *viper.Viper,
) Terraformer {
	return &ForkTerraformer{
		Logger:        logger,
		Daemon:        daemon,
		PlanConfirmer: planner,
		Terraform: func() *exec.Cmd {
			return exec.Command("/usr/local/bin/terraform")
		},
		Viper: viper,
	}
}

func (t *ForkTerraformer) WithDaemon(daemon daemon.Daemon) Terraformer {
	return &ForkTerraformer{
		Logger: t.Logger,
		Daemon: daemon,
	}
}

func (t *ForkTerraformer) Execute(ctx context.Context, release api.Release, step api.Terraform) error {

	assetsPath := filepath.Join("/tmp", "ship-terraform", version.RunAtEpoch, "asset")

	if err := t.init(assetsPath); err != nil {
		return errors.Wrapf(err, "init %s", assetsPath)
	}

	_, err := t.plan(assetsPath)
	if err != nil {
		return errors.Wrapf(err, "create plan for %s", assetsPath)
	}
	// create plan, save to state
	// push infra plan step
	// maybe exit
	// set progress applying

	fakePlan := "We're gonna make you some servers"

	if !viper.GetBool("terraform-yes") {
		shouldApply, err := t.PlanConfirmer.ConfirmPlan(ctx, fakePlan, release)
		if err != nil {
			return errors.Wrapf(err, "confirm plan for %s", assetsPath)
		}

		if !shouldApply {
			return nil
		}
	}

	// next: apply plan
	return nil
}

func (t *ForkTerraformer) init(assetsPath string) error {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "init"))

	cmd := t.Terraform()
	cmd.Args = append(cmd.Args, "init", "-input=false")
	cmd.Dir = assetsPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	debug.Log("stdout", string(out))
	debug.Log("stderr", stderr.String())
	if err != nil {
		return errors.Wrap(err, "exec terraform init")
	}

	return nil
}

func (t *ForkTerraformer) plan(modulePath string) (string, error) {
	// we really shouldn't write plan to a file, but this will do for now
	planOut := path.Join("tmp", "ship-terraform", version.RunAtEpoch, "plan")
	return planOut, nil
}
