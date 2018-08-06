package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"text/template"
	"time"

	"github.com/buildkite/terminal"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/viper"
)

type ForkKubectl struct {
	Logger  log.Logger
	Daemon  daemon.Daemon
	Kubectl func() *exec.Cmd
	Viper   *viper.Viper
}

func NewKubectl(
	logger log.Logger,
	daemon daemon.Daemon,
	viper *viper.Viper,
) lifecycle.Kubectl {
	return &ForkKubectl{
		Logger: logger,
		Daemon: daemon,
		Kubectl: func() *exec.Cmd {
			cmd := exec.Command("kubectl")
			cmd.Dir = constants.InstallerPrefixPath
			return cmd
		},
		Viper: viper,
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

	cmd := k.Kubectl()
	cmd.Args = append(cmd.Args, "apply", "-f", step.Path)
	if step.Kubeconfig != "" {
		cmd.Args = append(cmd.Args, "--kubeconfig", builtKubePath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	k.Daemon.SetProgress(daemon.StringProgress("kubectl", "applying kubernetes yaml with kubectl"))
	doneCh := make(chan struct{})
	messageCh := make(chan daemon.Message)
	go k.Daemon.PushStreamStep(ctx, messageCh)

	stderrString := ""
	stdoutString := ""
	go func() {
		for true {
			select {
			case <-time.After(time.Second):
				newStderr := stderr.String()
				newStdout := stdout.String()

				if newStderr != stderrString || newStdout != stdoutString {
					stderrString = newStderr
					stdoutString = newStdout
					messageCh <- daemon.Message{
						Contents:    ansiToHTML(stdoutString, stderrString),
						TrustedHTML: true,
					}
				}
			case <-doneCh:
				stderrString = stderr.String()
				stdoutString = stdout.String()
				close(messageCh)
				doneCh <- struct{}{}
				return
			}
		}
	}()

	err := cmd.Run()

	doneCh <- struct{}{}
	<-doneCh
	debug.Log("stdout", stdoutString)
	debug.Log("stderr", stderrString)

	if err != nil {
		stderrString = fmt.Sprintf(`Error: %s
stderr: %s`, err.Error(), stderrString)
	}

	k.Daemon.PushMessageStep(
		ctx,
		daemon.Message{
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
	debug := level.Debug(log.With(k.Logger, "struct", "daemonmessenger", "method", "message.confirm.await"))
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
			debug.Log("event", "message.confirmed")
			return nil
		case <-time.After(10 * time.Second):
			debug.Log("waitingFor", "message.confirmed")
		}
	}
}

type builderContext struct {
	logger log.Logger
	viper  *viper.Viper
	daemon daemon.Daemon
}

func (ctx builderContext) FuncMap() template.FuncMap {
	debug := level.Debug(log.With(ctx.logger, "step.type", "render", "render.phase", "template"))

	configFunc := func(name string) interface{} {
		configItemValue := ctx.viper.Get(name)
		if configItemValue == "" {
			debug.Log("event", "template.missing", "func", "config", "requested", name)
			return ""
		}
		return configItemValue
	}

	configItemFunc := func(name string) interface{} {
		if ctx.daemon == nil {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name)
			return ""
		}
		configItemValue, ok := ctx.daemon.GetCurrentConfig()[name]
		if !ok {
			debug.Log("event", "daemon.missing", "func", "ConfigOption", "requested", name)
		}
		return configItemValue
	}

	return map[string]interface{}{
		"config":       configFunc,
		"ConfigOption": configItemFunc,
	}
}

func (k *ForkKubectl) getBuilder(meta api.ReleaseMetadata) templates.Builder {
	builderBuilder := templates.NewBuilderBuilder(k.Logger)

	builder := builderBuilder.NewBuilder(
		builderBuilder.NewStaticContext(),
		builderContext{
			logger: k.Logger,
			viper:  k.Viper,
			daemon: k.Daemon,
		},
		&templates.InstallationContext{
			Meta:  meta,
			Viper: k.Viper,
		},
	)
	return builder
}
