package terraform

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/spf13/viper"
)

const tfSep = "------------------------------------------------------------------------"
const tfNoChanges = "No changes. Infrastructure is up-to-date."

type Terraformer interface {
	Execute(ctx context.Context, release api.Release, step api.Terraform) error
}

type ForkTerraformer struct {
	Logger        log.Logger
	Daemon        daemon.Daemon
	PlanConfirmer tfplan.PlanConfirmer
	Terraform     func(string) *exec.Cmd
	Viper         *viper.Viper
	dir           string
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
		Terraform: func(cmdPath string) *exec.Cmd {
			cmd := exec.Command("terraform")
			cmd.Dir = path.Join(constants.InstallerPrefixPath, cmdPath)
			return cmd
		},
		Viper: viper,
	}
}

func (t *ForkTerraformer) Execute(ctx context.Context, release api.Release, step api.Terraform) error {
	t.dir = step.Path

	if err := t.init(); err != nil {
		return errors.Wrap(err, "init")
	}

	plan, hasChanges, err := t.plan()
	if err != nil {
		return errors.Wrap(err, "plan")
	}
	if !hasChanges {
		return nil
	}

	if !viper.GetBool("terraform-yes") {
		shouldApply, err := t.PlanConfirmer.ConfirmPlan(ctx, ansiToHTML(plan), release)
		if err != nil {
			return errors.Wrap(err, "confirm plan")
		}

		if !shouldApply {
			return nil
		}
	}

	// capacity is whatever's required for tests to proceed
	applyMsgs := make(chan daemon.Message, 20)

	// returns when the applyMsgs channel closes
	go t.Daemon.PushStreamStep(ctx, applyMsgs)

	// blocks until all of stdout/stderr has been sent on applyMsgs channel
	html, err := t.apply(applyMsgs)
	close(applyMsgs)
	if err != nil {
		t.Daemon.PushMessageStep(
			ctx,
			daemon.Message{
				Contents:    html,
				TrustedHTML: true,
				Level:       "error",
			},
			failedApplyActions(),
		)
		retry := <-t.Daemon.TerraformConfirmedChan()
		t.Daemon.CleanPreviousStep()
		if retry {
			return t.Execute(ctx, release, step)
		}
		return errors.Wrap(err, "apply")
	}

	if !viper.GetBool("terraform-yes") {
		t.Daemon.PushMessageStep(
			ctx,
			daemon.Message{
				Contents:    html,
				TrustedHTML: true,
			},
			daemon.MessageActions(),
		)
		<-t.Daemon.MessageConfirmedChan()
	}

	return nil
}

func (t *ForkTerraformer) init() error {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "init"))

	cmd := t.Terraform(t.dir)
	cmd.Args = append(cmd.Args, "init", "-input=false")

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

// plan returns a human readable plan and a changes-required flag
func (t *ForkTerraformer) plan() (string, bool, error) {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "plan"))
	warn := level.Warn(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "plan"))

	// we really shouldn't write plan to a file, but this will do for now
	cmd := t.Terraform(t.dir)
	cmd.Args = append(cmd.Args, "plan", "-input=false", "-out=plan")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	debug.Log("stdout", string(out))
	debug.Log("stderr", stderr.String())
	if err != nil {
		return "", false, errors.Wrap(err, string(out)+"\n"+stderr.String())
		// return "", false, errors.Wrap(err, "exec terraform plan")
	}
	plan := string(out)

	if strings.Contains(plan, tfNoChanges) {
		debug.Log("changes", false)
		return "", false, nil
	}
	debug.Log("changes", true)

	// Drop 1st and 3rd sections with notes on state and how to apply
	sections := strings.Split(plan, tfSep)
	if len(sections) != 3 {
		warn.Log("plan.output.sections", len(sections))
		return plan, true, nil
	}

	return sections[1], true, nil
}

// apply returns the full stdout and stderr rendered as HTML
func (t *ForkTerraformer) apply(msgs chan<- daemon.Message) (string, error) {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "apply"))

	cmd := t.Terraform(t.dir)
	cmd.Args = append(cmd.Args, "apply", "-input=false", "-auto-approve=true", "plan")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.Wrap(err, "get stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, "get stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "command start")
	}

	// something to show while waiting for output
	msgs <- daemon.Message{
		Contents:    ansiToHTML("terraform apply"),
		TrustedHTML: true,
	}

	var accm string
	var readErr error
	var mtx sync.Mutex
	var wg sync.WaitGroup

	wg.Add(2)

	var pushAccmHTML = func(r io.Reader, name string) {
		defer wg.Done()

		b := make([]byte, 4096)
		for {
			n, err := r.Read(b)
			if n > 0 {
				latest := string(b[0:n])
				debug.Log(name, latest)
				mtx.Lock()
				accm += latest
				msg := daemon.Message{
					Contents:    ansiToHTML(accm),
					TrustedHTML: true,
				}
				msgs <- msg
				mtx.Unlock()
			}
			if err == io.EOF {
				return
			}
			if err != nil {
				mtx.Lock()
				readErr = errors.Wrapf(err, "read %s", name)
				mtx.Unlock()
				return
			}
		}
	}

	go pushAccmHTML(stdout, "stdout")
	go pushAccmHTML(stderr, "stderr")

	wg.Wait()

	if readErr != nil {
		return "", readErr
	}

	err = cmd.Wait()

	return ansiToHTML(accm), errors.Wrap(err, "command wait")
}

func failedApplyActions() []daemon.Action {
	return []daemon.Action{
		{
			ButtonType:  "primary",
			Text:        "Retry",
			LoadingText: "Retrying",
			OnClick: daemon.ActionRequest{
				URI:    "/terraform/apply",
				Method: "POST",
			},
		},
	}
}
