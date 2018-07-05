package helm

import (
	"fmt"
	"os/exec"
	"strings"

	"io/ioutil"

	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Templater is something that can consume and render a helm chart pulled by ship.
// the chart should already be present at the specified path.
type Templater interface {
	Template(
		chartRoot string,
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
	) error
}

var releaseNameRegex = regexp.MustCompile("[^a-zA-Z0-9\\-]")

// ForkTemplater implements Templater by forking out to an embedded helm binary
// and creating the chart in place
type ForkTemplater struct {
	Helm           func() *exec.Cmd
	Logger         log.Logger
	FS             afero.Afero
	BuilderBuilder *templates.BuilderBuilder
}

func (f *ForkTemplater) Template(
	chartRoot string,
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
) error {
	debug := level.Debug(log.With(f.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "helm", "dest", asset.Dest, "description", asset.Description))

	debug.Log("event", "mkdirall.attempt", "dest", asset.Dest, "basePath", asset.Dest)
	if err := f.FS.MkdirAll(asset.Dest, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "err", err, "basePath", asset.Dest)
		return errors.Wrapf(err, "write directory to %s", asset.Dest)
	}

	releaseName := strings.ToLower(fmt.Sprintf("%s", meta.ChannelName))
	releaseName = releaseNameRegex.ReplaceAllLiteralString(releaseName, "-")
	debug.Log("event", "releasename.resolve", "releasename", releaseName)

	// initialize command
	cmd := f.Helm()
	cmd.Args = append(
		cmd.Args,
		"template", chartRoot,
		"--output-dir", asset.Dest,
		"--name", releaseName,
	)

	if asset.HelmOpts != nil {
		cmd.Args = append(cmd.Args, asset.HelmOpts...)
	}

	if asset.Values != nil {
		for key, value := range asset.Values {
			cmd.Args = append(cmd.Args, "--set")
			cmd.Args = append(cmd.Args, fmt.Sprintf("%s=%s", key, value))
		}
	}

	stdout, stderr, err := f.fork(cmd)

	if err != nil {
		debug.Log("event", "cmd.err")
		if exitError, ok := err.(*exec.ExitError); ok && !exitError.Success() {
			return errors.Errorf(`execute helm: %s: stdout: "%s"; stderr: "%s";`, exitError.Error(), stdout, stderr)
		}
		return errors.Wrap(err, "execute helm")
	}

	// todo link up stdout/stderr debug logs
	return nil
}

func (f *ForkTemplater) fork(cmd *exec.Cmd) ([]byte, []byte, error) {
	debug := level.Debug(log.With(f.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "helm"))
	debug.Log("event", "cmd.run", "base", cmd.Path, "args", strings.Join(cmd.Args, " "))

	var stdout, stderr []byte
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "pipe stdout")
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "pipe stderr")
	}

	debug.Log("event", "cmd.start")
	err = cmd.Start()
	if err != nil {
		return stdout, stderr, errors.Wrap(err, "start cmd")
	}
	debug.Log("event", "cmd.started")

	stdout, err = ioutil.ReadAll(stdoutReader)
	if err != nil {
		debug.Log("event", "stdout.read.fail", "err", err)
		return stdout, stderr, errors.Wrap(err, "read stdout")
	}
	debug.Log("event", "stdout.read", "value", string(stdout))

	stderr, err = ioutil.ReadAll(stderrReader)
	if err != nil {
		debug.Log("event", "stderr.read.fail", "err", err)
		return stdout, stderr, errors.Wrap(err, "read stderr")
	}
	debug.Log("event", "stderr.read", "value", string(stderr))

	debug.Log("event", "cmd.wait")
	err = cmd.Wait()
	debug.Log("event", "cmd.waited")

	debug.Log("event", "cmd.streams.read.done")

	return stdout, stderr, err
}

// NewTemplater returns a configured Templater. For now we just always fork
func NewTemplater(
	logger log.Logger,
	fs afero.Afero,
	builderBuilder *templates.BuilderBuilder,
) Templater {
	return &ForkTemplater{
		Helm: func() *exec.Cmd {
			return exec.Command("/usr/local/bin/helm")
		},
		Logger:         logger,
		FS:             fs,
		BuilderBuilder: builderBuilder,
	}
}
