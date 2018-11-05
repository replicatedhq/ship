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
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type DaemonlessTerraformer struct {
	Logger         log.Logger
	PlanConfirmer  tfplan.PlanConfirmer
	Terraform      func(string) *exec.Cmd
	Status         daemontypes.StatusReceiver
	StateManager   state.Manager
	Viper          *viper.Viper
	FS             afero.Afero
	BuilderBuilder *templates.BuilderBuilder

	// exposed for testing
	StateRestorer stateRestorer
	StateSaver    stateSaver
}

func NewDaemonlessTerraformer(
	logger log.Logger,
	planner tfplan.PlanConfirmer,
	viper *viper.Viper,
	fs afero.Afero,
	statemanager state.Manager,
	builderBuilder *templates.BuilderBuilder,
) lifecycle.Terraformer {
	terraformPath := viper.GetString("terraform-exec-path")
	return &DaemonlessTerraformer{
		Logger:        logger,
		PlanConfirmer: planner,
		Terraform: func(cmdPath string) *exec.Cmd {
			cmd := exec.Command(terraformPath)
			cmd.Dir = cmdPath
			return cmd
		},
		Viper:          viper,
		FS:             fs,
		BuilderBuilder: builderBuilder,
		StateManager:   statemanager,
		StateSaver:     persistState,
		StateRestorer:  restoreState,
	}
}

func (t *DaemonlessTerraformer) WithStatusReceiver(
	statusReceiver daemontypes.StatusReceiver,
) lifecycle.Terraformer {
	return &DaemonlessTerraformer{
		Logger:         t.Logger,
		PlanConfirmer:  t.PlanConfirmer.WithStatusReceiver(statusReceiver),
		Terraform:      t.Terraform,
		Viper:          t.Viper,
		FS:             t.FS,
		BuilderBuilder: t.BuilderBuilder,
		StateManager:   t.StateManager,
		StateSaver:     t.StateSaver,
		StateRestorer:  t.StateRestorer,

		Status: statusReceiver,
	}
}

func (t *DaemonlessTerraformer) Execute(ctx context.Context, release api.Release, step api.Terraform, confirmedChan chan bool) error {
	debug := level.Debug(log.With(t.Logger, "struct", "ForkTerraformer", "method", "execute"))
	renderRoot := release.FindRenderRoot()
	dir := path.Join(renderRoot, step.Path)

	builder, err := t.BuilderBuilder.BaseBuilder(release.Metadata)
	if err != nil {
		return errors.Wrapf(err, "get builder")
	}

	if err := t.FS.MkdirAll(dir, 0755); err != nil {
		return errors.Wrapf(err, "mkdirall %s", dir)
	}

	debug.Log("event", "terraform.state.restore")
	if err := t.StateRestorer(debug, t.FS, t.StateManager, dir); err != nil {
		return errors.Wrapf(err, "restore terraform state")
	}

	if err := t.init(dir); err != nil {
		return errors.Wrap(err, "init")
	}

	plan, hasChanges, err := t.plan(dir)
	if err != nil {
		return errors.Wrap(err, "plan")
	}
	if !hasChanges {
		return nil
	}

	if !viper.GetBool("terraform-yes") {
		shouldApply, err := t.PlanConfirmer.ConfirmPlan(ctx, ansiToHTML(plan), release, confirmedChan)
		if err != nil {
			return errors.Wrap(err, "confirm plan")
		}

		if !shouldApply {
			return nil
		}
	}

	// capacity is whatever's required for tests to proceed
	applyMsgs := make(chan daemontypes.Message, 20)

	// returns when the applyMsgs channel closes
	go t.Status.PushStreamStep(ctx, applyMsgs)

	// blocks until all of stdout/stderr has been sent on applyMsgs channel
	html, err := t.apply(dir, applyMsgs)

	close(applyMsgs)
	if err != nil {
		t.Status.PushMessageStep(
			ctx,
			daemontypes.Message{
				Contents:    html,
				TrustedHTML: true,
				Level:       "error",
			},
			failedApplyActions(),
		)
		retry := <-confirmedChan
		if retry {
			return t.Execute(ctx, release, step, confirmedChan)
		}
		return errors.Wrap(err, "apply")
	}

	if gkeAsset := release.FindGKEAsset(); gkeAsset != nil {
		debug.Log("create kube config")
		if err := t.createKubeConfig(dir, builder, gkeAsset); err != nil {
			return errors.Wrap(err, "create kubeconfig for gke")
		}
	}

	if !viper.GetBool("terraform-yes") {
		t.Status.PushMessageStep(
			ctx,
			daemontypes.Message{
				Contents:    html,
				TrustedHTML: true,
			},
			finishMessageActions(),
		)
		<-confirmedChan
	}

	if err := t.StateSaver(debug, t.FS, t.StateManager, dir); err != nil {
		return errors.Wrapf(err, "persist terraform state")
	}

	return nil
}

func (t *DaemonlessTerraformer) init(dir string) error {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "init"))

	cmd := t.Terraform(dir)
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
func (t *DaemonlessTerraformer) plan(dir string) (string, bool, error) {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "plan"))
	warn := level.Warn(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "plan"))

	// we really shouldn't write plan to a file, but this will do for now
	cmd := t.Terraform(dir)
	cmd.Args = append(cmd.Args, "plan", "-input=false", "-out=plan.tfplan")

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
func (t *DaemonlessTerraformer) apply(dir string, msgs chan<- daemontypes.Message) (string, error) {
	debug := level.Debug(log.With(t.Logger, "step.type", "terraform", "terraform.phase", "apply"))

	cmd := t.Terraform(dir)
	cmd.Args = append(cmd.Args, "apply", "-input=false", "-auto-approve=true", "plan.tfplan")

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
	msgs <- daemontypes.Message{
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
				msg := daemontypes.Message{
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

func finishMessageActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			ButtonType:  "primary",
			Text:        "Confirm",
			LoadingText: "Confirming",
			OnClick: daemontypes.ActionRequest{
				URI:    "/terraform/apply",
				Method: "POST",
			},
		},
	}
}
