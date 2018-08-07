package helm

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/replicatedhq/ship/pkg/constants"

	"io/ioutil"

	"regexp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Templater is something that can consume and render a helm chart pulled by ship.
// the chart should already be present at the specified path.
type Templater interface {
	Template(
		chartRoot string,
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
		configGroups []libyaml.ConfigGroup,
		templateContext map[string]interface{},
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
	Viper          *viper.Viper
}

func (f *ForkTemplater) Template(
	chartRoot string,
	asset api.HelmAsset,
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
) error {
	debug := level.Debug(
		log.With(
			f.Logger,
			"step.type", "render",
			"render.phase", "execute",
			"asset.type", "helm",
			"chartRoot", chartRoot,
			"dest", asset.Dest,
			"description", asset.Description,
		),
	)
	debug.Log("event", "mkdirall.attempt", "helmtempdir", constants.RenderedHelmTempPath, "dest", asset.Dest)
	if err := f.FS.MkdirAll(constants.RenderedHelmTempPath, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "err", err, "helmtempdir", constants.RenderedHelmTempPath)
		return errors.Wrapf(err, "write tmp directory to %s", constants.RenderedHelmTempPath)
	}
	defer f.FS.RemoveAll(constants.RenderedHelmTempPath)

	releaseName := strings.ToLower(fmt.Sprintf("%s", meta.ChannelName))
	releaseName = releaseNameRegex.ReplaceAllLiteralString(releaseName, "-")
	debug.Log("event", "releasename.resolve", "releasename", releaseName)

	// initialize command
	cmd := f.Helm()
	cmd.Args = append(
		cmd.Args,
		"template", chartRoot,
		"--output-dir", constants.RenderedHelmTempPath,
		"--name", releaseName,
	)

	if asset.HelmOpts != nil {
		cmd.Args = append(cmd.Args, asset.HelmOpts...)
	}

	args, err := f.appendHelmValues(configGroups, templateContext, asset)
	if err != nil {
		return errors.Wrap(err, "build helm values")
	}
	cmd.Args = append(cmd.Args, args...)

	err = f.helmInitClient(chartRoot)
	if err != nil {
		return errors.Wrap(err, "init helm client")
	}

	err = f.helmDependencyUpdate(chartRoot)
	if err != nil {
		return errors.Wrap(err, "update helm dependencies")
	}

	stdout, stderr, err := f.fork(cmd)
	if err != nil {
		debug.Log("event", "cmd.err")
		if exitError, ok := err.(*exec.ExitError); ok && !exitError.Success() {
			return errors.Errorf(`execute helm: %s: stdout: "%s"; stderr: "%s";`, exitError.Error(), stdout, stderr)
		}
		return errors.Wrap(err, "execute helm")
	}

	// In app mode, copy the first found directory in RenderedHelmTempPath to dest
	if f.Viper.GetBool("is-app") {
		files, err := f.FS.ReadDir(constants.RenderedHelmTempPath)
		if err != nil {
			return errors.Wrap(err, "failed to read templates dir")
		}

		firstFoundFile := files[0]
		if !firstFoundFile.IsDir() {
			return errors.New(fmt.Sprintf("unable to find rendered chart, found file %s instead", firstFoundFile.Name()))
		}

		renderedChartDir := path.Join(constants.RenderedHelmTempPath, firstFoundFile.Name())
		if err := f.FS.Rename(renderedChartDir, asset.Dest); err != nil {
			return errors.Wrap(err, "failed to move rendered chart dir")
		}

		return nil
	}

	subChartsDirName := "charts"
	tempRenderedChartDir := path.Join(constants.RenderedHelmTempPath, meta.HelmChartMetadata.Name)
	tempRenderedChartTemplatesDir := path.Join(tempRenderedChartDir, "templates")
	tempRenderedSubChartsDir := path.Join(tempRenderedChartDir, subChartsDirName)

	if baseDirExists, err := f.FS.DirExists(constants.RenderedHelmPath); err == nil && baseDirExists {
		// remove base dir iff it is a dir and it exists
		if err := f.FS.RemoveAll(constants.RenderedHelmPath); err != nil {
			return errors.Wrap(err, "failed to remove base dir")
		}
		debug.Log("event", "remove base directory")
	}

	debug.Log("event", "rename")
	if templatesDirExists, err := f.FS.IsDir(tempRenderedChartTemplatesDir); err == nil && templatesDirExists {
		if err := f.FS.Rename(tempRenderedChartTemplatesDir, asset.Dest); err != nil {
			return errors.Wrap(err, "failed to rename templates dir")
		}
	} else {
		debug.Log("event", "rename", "folder", tempRenderedChartTemplatesDir, "message", "Folder does not exist")
	}

	if subChartsExist, err := f.FS.IsDir(tempRenderedSubChartsDir); err == nil && subChartsExist {
		if err := f.FS.Rename(tempRenderedSubChartsDir, path.Join(asset.Dest, subChartsDirName)); err != nil {
			return errors.Wrap(err, "failed to rename subcharts dir")
		}
	} else {
		debug.Log("event", "rename", "folder", tempRenderedSubChartsDir, "message", "Folder does not exist")
	}

	debug.Log("event", "temphelmvalues.remove", "path", constants.TempHelmValuesPath)
	if err := f.FS.RemoveAll(constants.TempHelmValuesPath); err != nil {
		return errors.Wrap(err, "failed to remove Helm values tmp dir")
	}

	// todo link up stdout/stderr debug logs
	return nil
}

func (f *ForkTemplater) appendHelmValues(
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
	asset api.HelmAsset,
) ([]string, error) {
	var cmdArgs []string
	configCtx, err := f.BuilderBuilder.NewConfigContext(configGroups, templateContext)
	if err != nil {
		return nil, errors.Wrap(err, "create config context")
	}
	builder := f.BuilderBuilder.NewBuilder(
		f.BuilderBuilder.NewStaticContext(),
		configCtx,
	)
	if asset.Values != nil {
		for key, value := range asset.Values {
			args, err := appendHelmValue(value, builder, cmdArgs, key)
			if err != nil {
				return nil, errors.Wrapf(err, "append helm value %s", key)
			}
			cmdArgs = append(cmdArgs, args...)
		}
	}
	return cmdArgs, nil
}

func appendHelmValue(
	value interface{},
	builder templates.Builder,
	args []string,
	key string,
) ([]string, error) {
	stringValue, ok := value.(string)
	if !ok {
		args = append(args, "--set")
		args = append(args, fmt.Sprintf("%s=%s", key, value))
		return args, nil
	}

	renderedValue, err := builder.String(stringValue)
	if err != nil {
		return nil, errors.Wrapf(err, "render value for %s", key)
	}
	args = append(args, "--set")
	args = append(args, fmt.Sprintf("%s=%s", key, renderedValue))
	return args, nil
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
func (f *ForkTemplater) helmDependencyUpdate(chartRoot string) error {
	debug := level.Debug(log.With(f.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "helm", "render.step", "helm.dependencyUpdate"))
	cmd := f.Helm()
	cmd.Args = append(cmd.Args,
		"dependency",
		"update",
		chartRoot,
	)

	debug.Log("event", "helm.update", "args", fmt.Sprintf("%v", cmd.Args))

	stdout, stderr, err := f.fork(cmd)

	if err != nil {
		debug.Log("event", "cmd.err")
		if exitError, ok := err.(*exec.ExitError); ok && !exitError.Success() {
			return errors.Errorf(`execute helm dependency update: %s: stdout: "%s"; stderr: "%s";`, exitError.Error(), stdout, stderr)
		}
		return errors.Wrap(err, "execute helm dependency update")
	}

	return nil
}

func (f *ForkTemplater) helmInitClient(chartRoot string) error {
	debug := level.Debug(log.With(f.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "helm"))
	cmd := f.Helm()
	cmd.Args = append(cmd.Args,
		"init",
		"--client-only",
	)

	debug.Log("event", "helm.initClient", "args", fmt.Sprintf("%v", cmd.Args))

	stdout, stderr, err := f.fork(cmd)

	if err != nil {
		debug.Log("event", "cmd.err")
		if exitError, ok := err.(*exec.ExitError); ok && !exitError.Success() {
			return errors.Errorf(`execute helm dependency update: %s: stdout: "%s"; stderr: "%s";`, exitError.Error(), stdout, stderr)
		}
		return errors.Wrap(err, "execute helm dependency update")
	}

	return nil
}

// NewTemplater returns a configured Templater. For now we just always fork
func NewTemplater(
	logger log.Logger,
	fs afero.Afero,
	builderBuilder *templates.BuilderBuilder,
	viper *viper.Viper,
) Templater {
	return &ForkTemplater{
		Helm: func() *exec.Cmd {
			return exec.Command("/usr/local/bin/helm")
		},
		Logger:         logger,
		FS:             fs,
		BuilderBuilder: builderBuilder,
		Viper:          viper,
	}
}
