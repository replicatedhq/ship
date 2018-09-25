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
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/templates"
)

type DaemonlessKubectl struct {
	Logger         log.Logger
	Status         daemontypes.StatusReceiver
	BuilderBuilder *templates.BuilderBuilder
}

func NewDaemonlessKubectl(
	logger log.Logger,
	builderBuilder *templates.BuilderBuilder,
) lifecycle.KubectlApply {
	return &DaemonlessKubectl{
		Logger:         logger,
		BuilderBuilder: builderBuilder,
	}
}

func (d *DaemonlessKubectl) WithStatusReceiver(statusReceiver daemontypes.StatusReceiver) lifecycle.KubectlApply {
	return &DaemonlessKubectl{
		Logger:         d.Logger,
		BuilderBuilder: d.BuilderBuilder,
		Status:         statusReceiver,
	}
}

func (d *DaemonlessKubectl) Execute(ctx context.Context, release api.Release, step api.KubectlApply, confirmedChan chan bool) error {
	builder, err := d.BuilderBuilder.BaseBuilder(release.Metadata)
	if err != nil {
		return errors.Wrap(err, "get builder")
	}

	builtPath, _ := builder.String(step.Path)
	builtKubePath, _ := builder.String(step.Kubeconfig)

	debug := level.Debug(log.With(d.Logger, "step.type", "kubectl"))

	if builtPath == "" {
		return errors.New("A path to apply is required")
	}

	cmd := exec.Command("kubectl")
	cmd.Dir = constants.InstallerPrefixPath
	cmd.Args = append(cmd.Args, "apply", "-f", step.Path)
	if step.Kubeconfig != "" {
		cmd.Args = append(cmd.Args, "--kubeconfig", builtKubePath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	d.Status.SetProgress(daemontypes.StringProgress("kubectl", "applying kubernetes yaml with kubectl"))
	doneCh := make(chan struct{})
	messageCh := make(chan daemontypes.Message)
	go d.Status.PushStreamStep(ctx, messageCh)

	stderrString := ""
	stdoutString := ""

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for true {
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
				}
			case <-doneCh:
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

	return d.awaitMessageConfirmed(ctx, confirmedChan)
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
