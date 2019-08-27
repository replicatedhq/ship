package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/buildkite/terminal"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/templates"
)

type ForkKubectl struct {
	Logger         log.Logger
	Daemon         daemontypes.Daemon
	BuilderBuilder *templates.BuilderBuilder
}

func NewKubectl(
	logger log.Logger,
	daemon daemontypes.Daemon,
	builderBuilder *templates.BuilderBuilder,
) lifecycle.KubectlApply {
	return &ForkKubectl{
		Logger:         logger,
		Daemon:         daemon,
		BuilderBuilder: builderBuilder,
	}
}

// WithStatusReceiver is a no-op for the ForkKubectl implementation using Daemon
func (k *ForkKubectl) WithStatusReceiver(status daemontypes.StatusReceiver) lifecycle.KubectlApply {
	return &ForkKubectl{
		Logger:         k.Logger,
		Daemon:         k.Daemon,
		BuilderBuilder: k.BuilderBuilder,
	}
}

func (k *ForkKubectl) Execute(ctx context.Context, release api.Release, step api.KubectlApply, confirmedChan chan bool) error {
	builder, err := k.BuilderBuilder.BaseBuilder(release.Metadata)
	if err != nil {
		return errors.Wrap(err, "get builder")
	}

	builtPath, _ := builder.String(step.Path)
	builtKubePath, _ := builder.String(step.Kubeconfig)

	debug := level.Debug(log.With(k.Logger, "step.type", "kubectl"))

	if builtPath == "" {
		return errors.New("A path to apply is required")
	}

	cmd := exec.Command("kubectl")
	cmd.Dir = release.FindRenderRoot()
	cmd.Args = append(cmd.Args, "apply", "-f", step.Path)
	if step.Kubeconfig != "" {
		cmd.Args = append(cmd.Args, "--kubeconfig", builtKubePath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	k.Daemon.SetProgress(daemontypes.StringProgress("kubectl", "applying kubernetes yaml with kubectl"))
	doneCh := make(chan struct{})
	messageCh := make(chan daemontypes.Message)
	go k.Daemon.PushStreamStep(ctx, messageCh)

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

	k.Daemon.PushMessageStep(
		ctx,
		daemontypes.Message{
			Contents:    ansiToHTML(stdoutString, stderrString),
			TrustedHTML: true,
		},
		daemon.MessageActions(),
	)

	daemonExitedChan := k.Daemon.EnsureStarted(ctx, &release)

	return k.awaitMessageConfirmed(ctx, daemonExitedChan)
}

func ansiToHTML(output, errors string) string {
	outputHTML := terminal.Render([]byte(output))
	errorsHTML := terminal.Render([]byte(errors))
	return fmt.Sprintf(`<header>Output:</header>
<div class="term-container">%s</div>
<header>Errors:</header>
<div class="term-container">%s</div>`, outputHTML, errorsHTML)
}

func (k *ForkKubectl) awaitMessageConfirmed(ctx context.Context, daemonExitedChan chan error) error {
	debug := level.Debug(log.With(k.Logger, "struct", "daemonmessenger", "method", "kubectl.confirm.await"))
	for {
		select {
		case <-ctx.Done():
			debug.Log("event", "ctx.done")
			return ctx.Err()
		case err := <-daemonExitedChan:
			debug.Log("event", "daemon.exit")
			if err != nil {
				return err
			}
			return errors.New("daemon exited")
		case <-k.Daemon.MessageConfirmedChan():
			debug.Log("event", "kubectl.message.confirmed")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "kubectl.message.confirmed")
		}
	}
}
