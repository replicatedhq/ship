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
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

type ForkKubectl struct {
	Logger log.Logger
	Daemon daemontypes.Daemon
	Viper  *viper.Viper
}

func NewKubectl(
	logger log.Logger,
	daemon daemontypes.Daemon,
	viper *viper.Viper,
) lifecycle.Kubectl {
	return &ForkKubectl{
		Logger: logger,
		Daemon: daemon,
		Viper:  viper,
	}
}

func (k *ForkKubectl) Execute(ctx context.Context, release api.Release, step api.Kubectl) error {
	builder := k.getBuilder(release.Metadata)
	builtPath, _ := builder.String(step.Path)
	builtKubePath, _ := builder.String(step.Kubeconfig)

	debug := level.Debug(log.With(k.Logger, "step.type", "kubectl"))

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

	k.Daemon.SetProgress(daemontypes.StringProgress("kubectl", "applying kubernetes yaml with kubectl"))
	doneCh := make(chan struct{})
	messageCh := make(chan daemontypes.Message)
	go k.Daemon.PushStreamStep(ctx, messageCh)

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

	err := cmd.Run()

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

func (k *ForkKubectl) getBuilder(meta api.ReleaseMetadata) templates.Builder {
	builderBuilder := templates.NewBuilderBuilder(k.Logger)

	builder := builderBuilder.NewBuilder(
		builderBuilder.NewStaticContext(),
		&templates.InstallationContext{
			Meta:  meta,
			Viper: k.Viper,
		},
		templates.ConfigCtx{
			ItemValues: k.Daemon.GetCurrentConfig(),
			Logger:     k.Logger,
		},
	)
	return builder
}
