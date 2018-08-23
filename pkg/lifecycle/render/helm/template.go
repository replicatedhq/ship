package helm

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/process"
	"github.com/replicatedhq/ship/pkg/util"

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

// NewTemplater returns a configured Templater that uses vendored libhelm to execute templating/etc
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
	renderDest := path.Join(constants.ShipPathInternalTmp, "chartrendered")
	err := f.FS.MkdirAll(renderDest, 0755)
	if err != nil {
		debug.Log("event", "mkdirall.fail", "err", err, "helmtempdir", renderDest)
		return errors.Wrapf(err, "create tmp directory in %s", constants.ShipPathInternalTmp)
	}

	releaseName := strings.ToLower(fmt.Sprintf("%s", meta.ReleaseName()))
	releaseName = releaseNameRegex.ReplaceAllLiteralString(releaseName, "-")
	debug.Log("event", "releasename.resolve", "releasename", releaseName)

	templateArgs := []string{
		"--output-dir", renderDest,
		"--name", releaseName,
	}

	if asset.HelmOpts != nil {
		templateArgs = append(templateArgs, asset.HelmOpts...)
	}

	debug.Log("event", "helm.init")
	if err := f.Commands.Init(); err != nil {
		return errors.Wrap(err, "init helm client")
	}

	debug.Log("event", "helm.dependency.update")
	if err := f.Commands.DependencyUpdate(chartRoot); err != nil {
		return errors.Wrap(err, "update helm dependencies")
	}

	if asset.ValuesFrom != nil && asset.ValuesFrom.Lifecycle != nil {
		tmpValuesPath := path.Join(constants.ShipPathInternalTmp, "values.yaml")
		defaultValuesPath := path.Join(chartRoot, "values.yaml")
		debug.Log("event", "writeTmpValues", "to", tmpValuesPath, "default", defaultValuesPath)
		if err := f.writeStateHelmValuesTo(tmpValuesPath, defaultValuesPath); err != nil {
			return errors.Wrapf(err, "copy state value to tmp directory", renderDest)
		}

		templateArgs = append(templateArgs,
			"--values",
			tmpValuesPath,
		)

	}

	if len(asset.Values) > 0 {
		args, err := f.appendHelmValues(configGroups, templateContext, asset)
		if err != nil {
			return errors.Wrap(err, "build helm values")
		}
		templateArgs = append(templateArgs, args...)
	}

	debug.Log("event", "helm.template")
	if err := f.Commands.Template(chartRoot, templateArgs); err != nil {
		debug.Log("event", "helm.template.err")
		return errors.Wrap(err, "execute helm")
	}

	tempRenderedChartDir, err := f.getTempRenderedChartDirectoryName(renderDest, meta)
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

func (f *LocalTemplater) getTempRenderedChartDirectoryName(renderRoot string, meta api.ReleaseMetadata) (string, error) {
	if meta.ShipAppMetadata.Name != "" {
		return path.Join(renderRoot, meta.ShipAppMetadata.Name), nil
	}

	return util.FindOnlySubdir(renderRoot, f.FS)
}

func (f *LocalTemplater) cleanUpAndOutputRenderedFiles(
	rootFs root.Fs,
	asset api.HelmAsset,
	tempRenderedChartDir string,
) error {
	debug := level.Debug(log.With(f.Logger, "method", "cleanUpAndOutputRenderedFiles"))

	subChartsDirName := "charts"
	tempRenderedChartTemplatesDir := path.Join(tempRenderedChartDir, "templates")
	tempRenderedSubChartsDir := path.Join(tempRenderedChartDir, subChartsDirName)

	debug.Log("event", "removeall", "path", constants.KustomizeBasePath) // todo fail if this exists
	if err := f.FS.RemoveAll(constants.KustomizeBasePath); err != nil {
		debug.Log("event", "removeall.fail", "path", constants.KustomizeBasePath)
		return errors.Wrap(err, "failed to remove rendered Helm values base dir")
	}

	debug.Log("event", "mkdirall", "path", asset.Dest)

	if err := rootFs.MkdirAll(asset.Dest, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "path", asset.Dest)
		return errors.Wrap(err, "failed to make asset destination base directory")
	}

	if templatesDirExists, err := f.FS.IsDir(tempRenderedChartTemplatesDir); err != nil || !templatesDirExists {
		return errors.Wrap(err, "unable to find tmp rendered chart")
	}

	debug.Log("event", "readdir", "folder", tempRenderedChartTemplatesDir)
	files, err := f.FS.ReadDir(tempRenderedChartTemplatesDir)
	if err != nil {
		debug.Log("event", "readdir.fail", "folder", tempRenderedChartTemplatesDir)
		return errors.Wrap(err, "failed to read temp rendered charts folder")
	}

	for _, file := range files {
		originalPath := path.Join(tempRenderedChartTemplatesDir, file.Name())
		renderedPath := path.Join(rootFs.RootPath, asset.Dest, file.Name())
		if err := f.FS.Rename(originalPath, renderedPath); err != nil {
			fileType := "file"
			if file.IsDir() {
				fileType = "directory"
			}
			return errors.Wrapf(err, "failed to rename %s at path %s", fileType, originalPath)
		}
	}

	if subChartsExist, err := rootFs.IsDir(tempRenderedSubChartsDir); err == nil && subChartsExist {
		if err := rootFs.Rename(tempRenderedSubChartsDir, path.Join(asset.Dest, subChartsDirName)); err != nil {
			return errors.Wrap(err, "failed to rename subcharts dir")
		}
	} else {
		debug.Log("event", "rename", "folder", tempRenderedSubChartsDir, "message", "Folder does not exist")
	}

	debug.Log("event", "removeall", "path", constants.TempHelmValuesPath)
	if err := f.FS.RemoveAll(constants.TempHelmValuesPath); err != nil {
		debug.Log("event", "removeall.fail", "path", constants.TempHelmValuesPath)
		return errors.Wrap(err, "failed to remove Helm values tmp dir")
	}

	return nil
}

// dest should be a path to a file, and its parent directory should already exist
// if there are no values in state, defaultValuesPath will be copied into dest
func (f *LocalTemplater) writeStateHelmValuesTo(dest string, defaultValuesPath string) error {
	debug := level.Debug(log.With(f.Logger, "step.type", "helmValues", "resolveHelmValues"))
	debug.Log("event", "tryLoadState")
	editState, err := f.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	helmValues := editState.CurrentHelmValues()
	defaultHelmValues := editState.CurrentHelmValuesDefaults()

	defaultValuesShippedWithChartBytes, err := f.FS.ReadFile(filepath.Join(constants.HelmChartPath, "values.yaml"))
	if err != nil {
		return errors.Wrapf(err, "read helm values from %s", filepath.Join(constants.HelmChartPath, "values.yaml"))
	}
	defaultValuesShippedWithChart := string(defaultValuesShippedWithChartBytes)

	if helmValues == "" {
		debug.Log("event", "values.load", "message", "No helm values in state; using values shipped with chart.")
		helmValues = defaultValuesShippedWithChart
	}
	if defaultHelmValues == "" {
		debug.Log("event", "values.load", "message", "No default helm values in state; using helm values from state.")
		defaultHelmValues = defaultValuesShippedWithChart
	}

	mergedValues, err := MergeHelmValues(defaultHelmValues, helmValues, defaultValuesShippedWithChart)
	if err != nil {
		return errors.Wrap(err, "merge helm values")
	}

	err = f.FS.MkdirAll(constants.TempHelmValuesPath, 0700)
	if err != nil {
		return errors.Wrapf(err, "make dir %s", constants.TempHelmValuesPath)
	}
	debug.Log("event", "writeTempValuesYaml", "dest", dest)
	err = f.FS.WriteFile(dest, []byte(mergedValues), 0644)
	if err != nil {
		return errors.Wrapf(err, "write values.yaml to %s", dest)
	}

	err = f.StateManager.SerializeHelmValues(mergedValues, string(defaultValuesShippedWithChartBytes))
	if err != nil {
		return errors.Wrapf(err, "serialize helm values to state")
	}

	return nil
}
