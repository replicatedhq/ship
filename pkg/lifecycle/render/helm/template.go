package helm

import (
	"fmt"
	"path"
	"strings"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/process"

	"regexp"

	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Templater is something that can consume and render a helm chart pulled by ship.
// the chart should already be present at the specified path.
type Templater interface {
	Template(
		chartRoot string,
		rootFs root.Fs,
		asset api.HelmAsset,
		meta api.ReleaseMetadata,
		configGroups []libyaml.ConfigGroup,
		templateContext map[string]interface{},
	) error
}

var releaseNameRegex = regexp.MustCompile("[^a-zA-Z0-9\\-]")

// LocalTemplater implements Templater by using the Commands interface
// from pkg/helm and creating the chart in place
type LocalTemplater struct {
	Commands       Commands
	Logger         log.Logger
	FS             afero.Afero
	BuilderBuilder *templates.BuilderBuilder
	Viper          *viper.Viper
	StateManager   state.Manager
	process        process.Process
}

func (f *LocalTemplater) Template(
	chartRoot string,
	rootFs root.Fs,
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

	debug.Log("event", "mkdirall.attempt")
	if err := rootFs.MkdirAll(constants.RenderedHelmTempPath, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "err", err, "helmtempdir", constants.RenderedHelmTempPath)
		return errors.Wrapf(err, "write tmp directory to %s", path.Join(rootFs.RootPath, constants.RenderedHelmTempPath))
	}
	defer rootFs.RemoveAll(constants.RenderedHelmTempPath)

	releaseName := strings.ToLower(fmt.Sprintf("%s", meta.ReleaseName()))
	releaseName = releaseNameRegex.ReplaceAllLiteralString(releaseName, "-")
	debug.Log("event", "releasename.resolve", "releasename", releaseName)

	templateArgs := []string{
		"--output-dir", path.Join(rootFs.RootPath, constants.RenderedHelmTempPath),
		"--name", releaseName,
	}

	if asset.HelmOpts != nil {
		templateArgs = append(templateArgs, asset.HelmOpts...)
	}

	args, err := f.appendHelmValues(configGroups, templateContext, asset)
	if err != nil {
		return errors.Wrap(err, "build helm values")
	}
	templateArgs = append(templateArgs, args...)

	debug.Log("event", "helm.init")
	if err := f.Commands.Init(); err != nil {
		return errors.Wrap(err, "init helm client")
	}

	debug.Log("event", "helm.dependency.update")
	err = f.Commands.DependencyUpdate(chartRoot)
	if err != nil {
		return errors.Wrap(err, "update helm dependencies")
	}

	if !viper.GetBool("is-app") {
		// HACKKK for ship init

		// todo move this, or refactor to share duped code from helmValues package
		if err := writeStateHelmValuesToChartTmpdir(f.Logger, f.StateManager, f.FS); err != nil {
			return errors.Wrapf(err, "copy state value to tmp directory", constants.RenderedHelmTempPath)
		}
	}

	debug.Log("event", "helm.template")
	if err := f.Commands.Template(chartRoot, templateArgs); err != nil {
		debug.Log("event", "helm.template.err")
		return errors.Wrap(err, "execute helm")
	}

	tempRenderedChartDir, err := f.getTempRenderedChartDirectoryName(rootFs, meta)
	if err != nil {
		return err
	}
	return f.cleanUpAndOutputRenderedFiles(rootFs, asset, tempRenderedChartDir)
}

func (f *LocalTemplater) appendHelmValues(
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

func (f *LocalTemplater) getTempRenderedChartDirectoryName(rootFs root.Fs, meta api.ReleaseMetadata) (string, error) {
	if meta.HelmChartMetadata.Name != "" {
		return path.Join(constants.RenderedHelmTempPath, meta.HelmChartMetadata.Name), nil
	}

	files, err := rootFs.ReadDir(constants.RenderedHelmTempPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read templates dir")
	}

	if len(files) == 0 {
		return "", errors.New("No files found in templates dir")
	}

	firstFoundFile := files[0]
	if !firstFoundFile.IsDir() {
		return "", errors.New(fmt.Sprintf("unable to find rendered chart, found file %s instead", firstFoundFile.Name()))
	}

	return path.Join(constants.RenderedHelmTempPath, firstFoundFile.Name()), nil
}

func (f *LocalTemplater) cleanUpAndOutputRenderedFiles(
	rootFs root.Fs,
	asset api.HelmAsset,
	tempRenderedChartDir string,
) error {
	debug := level.Debug(log.With(f.Logger, "method", "cleanUpAndOutputRenderedFiles"))

	tempRenderedChartTemplatesDir := path.Join(tempRenderedChartDir, "templates")

	debug.Log("event", "removeall", "path", constants.RenderedHelmPath)
	if err := f.FS.RemoveAll(constants.RenderedHelmPath); err != nil {
		debug.Log("event", "removeall.fail", "path", constants.RenderedHelmPath)
		return errors.Wrap(err, "failed to remove rendered Helm values base dir")
	}

	debug.Log("event", "mkdirall", "path", asset.Dest)
	if err := rootFs.MkdirAll(asset.Dest, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "path", asset.Dest)
		return errors.Wrap(err, "failed to make asset destination base directory")
	}

	if templatesDirExists, err := rootFs.IsDir(tempRenderedChartTemplatesDir); err == nil && templatesDirExists {
		debug.Log("event", "readdir.fail", "folder", tempRenderedChartTemplatesDir)
		files, err := rootFs.ReadDir(tempRenderedChartTemplatesDir)
		if err != nil {
			debug.Log("event", "readdir.fail", "folder", tempRenderedChartTemplatesDir)
			return errors.Wrap(err, "failed to read temp rendered charts folder")
		}
		for _, file := range files {
			originalPath := path.Join(tempRenderedChartTemplatesDir, file.Name())
			renderedPath := path.Join(asset.Dest, file.Name())
			if err := rootFs.Rename(originalPath, renderedPath); err != nil {
				fileType := "file"
				if file.IsDir() {
					fileType = "directory"
				}
				return errors.Wrapf(err, "failed to rename %s at path %s", fileType, originalPath)
			}
		}
	} else {
		return errors.Wrap(err, "unable to find tmp rendered chart")
	}

	debug.Log("event", "removeall", "path", constants.TempHelmValuesPath)
	if err := f.FS.RemoveAll(constants.TempHelmValuesPath); err != nil {
		debug.Log("event", "removeall.fail", "path", constants.TempHelmValuesPath)
		return errors.Wrap(err, "failed to remove Helm values tmp dir")
	}

	return nil
}

// NewTemplater returns a configured Templater. For now we just always fork
func NewTemplater(
	commands Commands,
	logger log.Logger,
	fs afero.Afero,
	builderBuilder *templates.BuilderBuilder,
	viper *viper.Viper,
	stateManager state.Manager,
) Templater {
	return &LocalTemplater{
		Commands:       commands,
		Logger:         logger,
		FS:             fs,
		BuilderBuilder: builderBuilder,
		Viper:          viper,
		StateManager:   stateManager,
		process:        process.Process{Logger: logger},
	}
}

// TODO duped from lifecycle/helmValues
func writeStateHelmValuesToChartTmpdir(logger log.Logger, manager state.Manager, fs afero.Afero) error {
	debug := level.Debug(log.With(logger, "step.type", "helmValues", "resolveHelmValues"))
	debug.Log("event", "tryLoadState")
	editState, err := manager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	helmValues := editState.CurrentHelmValues()
	if helmValues == "" {
		defaultValuesShippedWithChart := filepath.Join(constants.KustomizeHelmPath, "values.yaml")
		bytes, err := fs.ReadFile(defaultValuesShippedWithChart)
		if err != nil {
			return errors.Wrapf(err, "read helm values from %s", defaultValuesShippedWithChart)
		}
		helmValues = string(bytes)
	}
	debug.Log("event", "tryLoadState")
	err = fs.MkdirAll(constants.TempHelmValuesPath, 0700)
	if err != nil {
		return errors.Wrapf(err, "make dir %s", constants.TempHelmValuesPath)
	}
	debug.Log("event", "writeTempValuesYaml")
	err = fs.WriteFile(path.Join(constants.TempHelmValuesPath, "values.yaml"), []byte(helmValues), 0644)
	if err != nil {
		return errors.Wrapf(err, "write values.yaml to %s", constants.TempHelmValuesPath)
	}
	return nil
}
