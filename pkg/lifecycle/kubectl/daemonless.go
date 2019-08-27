package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
)

type DaemonlessKubectl struct {
	Logger         log.Logger
	Status         daemontypes.StatusReceiver
	StateManager   state.Manager
	BuilderBuilder *templates.BuilderBuilder
}

func NewDaemonlessKubectl(
	logger log.Logger,
	builderBuilder *templates.BuilderBuilder,
	statemanager state.Manager,
) lifecycle.KubectlApply {
	return &DaemonlessKubectl{
		Logger:         logger,
		BuilderBuilder: builderBuilder,
		StateManager:   statemanager,
	}
}

func (d *DaemonlessKubectl) WithStatusReceiver(statusReceiver daemontypes.StatusReceiver) lifecycle.KubectlApply {
	return &DaemonlessKubectl{
		Logger:         d.Logger,
		BuilderBuilder: d.BuilderBuilder,
		StateManager:   d.StateManager,
		Status:         statusReceiver,
	}
}

// TODO I need tests
func (d *DaemonlessKubectl) Execute(ctx context.Context, release api.Release, step api.KubectlApply, confirmedChan chan bool) error {
	debug := level.Debug(log.With(d.Logger, "step.type", "kubectl"))

	cmd, err := d.prepareCmd(release, step)
	if err != nil {
		return errors.Wrap(err, "failed to prepare command for daemonless kubectl execution")
	}

	debug.Log("event", "kubectl.execute", "args", fmt.Sprintf("%+v", cmd.Args))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	d.Status.SetProgress(daemontypes.StringProgress("kubectl", "applying kubernetes yaml with kubectl"))
	doneCh := make(chan struct{})
	messageCh := make(chan daemontypes.Message)
	go d.Status.PushStreamStep(ctx, messageCh)
	debug.Log("event", "kubectl.streamStep.pushed")

	stderrString := ""
	stdoutString := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for {
			select {
			case <-time.After(time.Second):
				newStderr := stderr.String()
				newStdout := stdout.String()

				if newStderr != stderrString || newStdout != stdoutString {
					stderrString = newStderr
					stdoutString = newStdout
					messageCh <- daemontypes.Message{
						Contents:    ansiToHTML(stdoutString, stderrString),
						TrustedHTML: true,
					}
					debug.Log("event", "kubectl.message.pushed")
				}
			case <-doneCh:
				debug.Log("event", "kubectl.doneCh")
				stderrString = stderr.String()
				stdoutString = stdout.String()
				close(messageCh)
				wg.Done()
				return
			}
		}
	}()

	err = cmd.Run()

	doneCh <- struct{}{}
	wg.Wait()

	debug.Log("stdout", stdoutString)
	debug.Log("stderr", stderrString)

	if err != nil {
		debug.Log("event", "kubectl.run.error", "err", err)
		stderrString = fmt.Sprintf(`Error: %s
	stderr: %s`, err.Error(), stderrString)
	}

	d.Status.PushMessageStep(
		ctx,
		daemontypes.Message{
			Contents:    ansiToHTML(stdoutString, stderrString),
			TrustedHTML: true,
		},
		confirmActions(),
	)
	debug.Log("event", "kubectl.outputs.pushed", "next", "confirmed.await")

	return d.awaitMessageConfirmed(ctx, confirmedChan)
}

func (d *DaemonlessKubectl) prepareCmd(release api.Release, step api.KubectlApply) (*exec.Cmd, error) {
	currState, err := d.StateManager.CachedState()
	if err != nil {
		return nil, errors.Wrap(err, "load state")
	}

	currentConfig, err := currState.CurrentConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get current config")
	}

	builder, err := d.BuilderBuilder.FullBuilder(release.Metadata, release.Spec.Config.V1, currentConfig)
	if err != nil {
		return nil, errors.Wrap(err, "get builder")
	}

	builtPath, err := builder.String(step.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "build apply path %s", step.Path)
	}
	builtKubePath, err := builder.String(step.Kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "build kubeconfig path %s", step.Kubeconfig)
	}

	if builtPath == "" {
		return nil, errors.New("A path to apply is required")
	}

	cmd := exec.Command("kubectl")
	cmd.Dir = release.FindRenderRoot()
	cmd.Args = append(cmd.Args, "apply", "-f", builtPath)
	if step.Kubeconfig != "" {
		cmd.Args = append(cmd.Args, "--kubeconfig", builtKubePath)
	}
	return cmd, nil
}

func (d *DaemonlessKubectl) awaitMessageConfirmed(ctx context.Context, confirmedChan chan bool) error {
	debug := level.Debug(log.With(d.Logger, "struct", "daemonlesskubectl", "method", "awaitMessageConfirmed"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return ctx.Err()
		case <-confirmedChan:
			debug.Log("event", "kubectl.message.confirmed")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "kubectl.message.confirmed")
		}
	}
}

func confirmActions() []daemontypes.Action {
	return []daemontypes.Action{
		{
			ButtonType:  "primary",
			Text:        "Confirm",
			LoadingText: "Confirming",
			OnClick: daemontypes.ActionRequest{
				URI:    "/kubectl/confirm",
				Method: "POST",
			},
		},
	}
}
