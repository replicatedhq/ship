package helm

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emosbaugh/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/process"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"k8s.io/helm/pkg/chartutil"
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

var arrayLineRegex = regexp.MustCompile(`^\s*(args|volumes):\s*$`)
var envLineRegex = regexp.MustCompile(`^\s*env:\s*$`)
var valueLineRegex = regexp.MustCompile(`^\s*value:\s*$`)

var nullValueLineRegex = regexp.MustCompile(`^(\s*value:)\s*null\s*$`)

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
		log.With(f.Logger,
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

	err := f.FS.RemoveAll(renderDest)
	if err != nil {
		debug.Log("event", "removeall.fail", "err", err, "helmtempdir", renderDest)
		return errors.Wrapf(err, "remove tmp directory in %s", constants.ShipPathInternalTmp)
	}

	err = f.FS.MkdirAll(renderDest, 0755)
	if err != nil {
		debug.Log("event", "mkdirall.fail", "err", err, "helmtempdir", renderDest)
		return errors.Wrapf(err, "create tmp directory in %s", constants.ShipPathInternalTmp)
	}

	state, err := f.StateManager.TryLoad()
	if err != nil {
		debug.Log("event", "tryloadState.fail", "err", err)
		return errors.Wrapf(err, "try load state")
	}

	versioned := state.Versioned()
	releaseName := versioned.CurrentReleaseName()
	debug.Log("event", "releasename.resolve.fromState", "releasename", releaseName)

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

	debug.Log("event", "helm.get.requirements")
	requirements, err := f.getChartRequirements(chartRoot)
	if err != nil {
		return errors.Wrap(err, "get chart requirements")
	}

	debug.Log("event", "helm.repo.add")
	absTempHelmHome, err := filepath.Abs(constants.InternalTempHelmHome)
	if err != nil {
		return errors.Wrap(err, "make absolute helm temp home")
	}

	depPaths, err := f.addDependencies(
		requirements.Dependencies,
		absTempHelmHome,
		chartRoot,
		asset,
	)
	if err != nil {
		return errors.Wrapf(err, "add requirements deps for %s", asset.Upstream)
	}

	debug.Log("event", "helm.dependency.update")
	if err := f.Commands.MaybeDependencyUpdate(chartRoot, requirements); err != nil {
		return errors.Wrapf(err, "update helm dependencies for %s", asset.Upstream)
	}

	if asset.ValuesFrom != nil {
		var valuesPath string
		defaultValuesPath := path.Join(chartRoot, "values.yaml")

		if asset.ValuesFrom.Path != "" {
			valuesPath = path.Join(asset.ValuesFrom.Path, "values.yaml")
		}

		debug.Log("event", "writeTmpValues", "to", valuesPath, "default", defaultValuesPath)
		if err := f.writeStateHelmValuesTo(valuesPath, defaultValuesPath); err != nil {
			return errors.Wrapf(err, "copy state value to tmp directory %s", renderDest)
		}
		templateArgs = append(templateArgs,
			"--values",
			valuesPath,
		)

		if asset.ValuesFrom.SaveToState {
			if err := f.writeMergedAndDefaultHelmValues(valuesPath, defaultValuesPath); err != nil {
				return errors.Wrap(err, "write merged and default helm values")
			}
		}
	}

	if len(asset.Values) > 0 {
		args, err := f.appendHelmValues(
			meta,
			configGroups,
			templateContext,
			asset,
		)
		if err != nil {
			return errors.Wrap(err, "build helm values")
		}
		templateArgs = append(templateArgs, args...)
	}

	namespace := versioned.CurrentNamespace()
	if len(namespace) > 0 {
		templateArgs = addArgIfNotPresent(templateArgs, "--namespace", namespace)
	} else {
		templateArgs = addArgIfNotPresent(templateArgs, "--namespace", "default")
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
	return f.cleanUpAndOutputRenderedFiles(rootFs, asset, tempRenderedChartDir, depPaths)
}

func (f *LocalTemplater) getChartRequirements(chartRoot string) (chartutil.Requirements, error) {
	requirements := chartutil.Requirements{}

	requirementsExists, err := f.FS.Exists(filepath.Join(chartRoot, "requirements.yaml"))
	if err != nil {
		return requirements, errors.Wrap(err, "check requirements yaml existence")
	}

	if !requirementsExists {
		return requirements, nil
	}

	requirementsB, err := f.FS.ReadFile(filepath.Join(chartRoot, "requirements.yaml"))
	if err != nil {
		return requirements, errors.Wrap(err, "read requirements yaml")
	}

	if err := yaml.Unmarshal(requirementsB, &requirements); err != nil {
		return requirements, errors.Wrap(err, "unmarshal requirements yaml")
	}

	return requirements, nil
}

// checks to see if the specified arg is present in the list. If it is not, adds it set to the specified value
func addArgIfNotPresent(existingArgs []string, newArg string, newDefault string) []string {
	for _, arg := range existingArgs {
		if arg == newArg {
			return existingArgs
		}
	}

	return append(existingArgs, newArg, newDefault)
}

func (f *LocalTemplater) appendHelmValues(
	meta api.ReleaseMetadata,
	configGroups []libyaml.ConfigGroup,
	templateContext map[string]interface{},
	asset api.HelmAsset,
) ([]string, error) {
	var cmdArgs []string
	builder, err := f.BuilderBuilder.FullBuilder(
		meta,
		configGroups,
		templateContext,
	)
	if err != nil {
		return nil, errors.Wrap(err, "initialize template builder")
	}

	if asset.Values != nil {
		for key, value := range asset.Values {
			args, err := appendHelmValue(value, *builder, cmdArgs, key)
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
	depPaths []string,
) error {
	debug := level.Debug(log.With(f.Logger, "method", "cleanUpAndOutputRenderedFiles"))

	subChartsDirName := "charts"
	tempRenderedChartTemplatesDir := path.Join(tempRenderedChartDir, "templates")
	tempRenderedSubChartsDir := path.Join(tempRenderedChartDir, subChartsDirName)

	if f.Viper.GetBool("rm-asset-dest") {
		debug.Log("event", "baseDir.rm", "path", asset.Dest)
		if err := f.FS.RemoveAll(asset.Dest); err != nil {
			return errors.Wrapf(err, "rm asset dest, remove %s", asset.Dest)
		}
	}

	debug.Log("event", "bailIfPresent", "path", asset.Dest)
	if err := util.BailIfPresent(f.FS, asset.Dest, f.Logger); err != nil {
		return err
	}

	debug.Log("event", "mkdirall", "path", asset.Dest)
	if err := rootFs.MkdirAll(asset.Dest, 0755); err != nil {
		debug.Log("event", "mkdirall.fail", "path", asset.Dest)
		return errors.Wrap(err, "failed to make asset destination base directory")
	}

	templatesDirExists, err := f.FS.DirExists(tempRenderedChartTemplatesDir)
	if err != nil || !templatesDirExists {
		// Sometimes the template dir doesn't exist
		debug.Log("event", "templateDirNotFound")
	}

	if err := f.validateGeneratedFiles(f.FS, tempRenderedChartDir); err != nil {
		return errors.Wrapf(err, "unable to validate chart dir")
	}

	if templatesDirExists {
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
	}

	if subChartsExist, err := rootFs.IsDir(tempRenderedSubChartsDir); err == nil && subChartsExist {
		debug.Log("event", "rename", "folder", tempRenderedSubChartsDir)
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

	for _, depPath := range depPaths {
		debug.Log("event", "removeall", "path", depPath)
		if err := f.FS.RemoveAll(depPath); err != nil {
			return errors.Wrapf(err, "failed to remove chart dep %s", depPath)
		}
	}

	return nil
}

func (f *LocalTemplater) writeMergedAndDefaultHelmValues(valuesPath, defaultValuesPath string) error {
	valuesB, err := f.FS.ReadFile(valuesPath)
	if err != nil {
		return errors.Wrapf(err, "read values path %s", valuesPath)
	}

	defaultValuesB, err := f.FS.ReadFile(defaultValuesPath)
	if err != nil {
		return errors.Wrapf(err, "read default values path %s", defaultValuesPath)
	}

	if err := f.StateManager.SerializeHelmValues(string(valuesB), string(defaultValuesB)); err != nil {
		return errors.Wrap(err, "serialize helm values")
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

	defaultValuesShippedWithChartBytes, err := f.FS.ReadFile(defaultValuesPath)
	if err != nil {
		return errors.Wrapf(err, "read helm values from %s", defaultValuesPath)
	}
	defaultValuesShippedWithChart := string(defaultValuesShippedWithChartBytes)

	if defaultHelmValues == "" {
		debug.Log("event", "values.load", "message", "No default helm values in state; using helm values from state.")
		defaultHelmValues = defaultValuesShippedWithChart
	}

	mergedValues, err := MergeHelmValues(defaultHelmValues, helmValues, defaultValuesShippedWithChart, false)
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

	return nil
}

// validate each file to make sure that it conforms to the yaml spec
// TODO replace this with an actual validation tool
func (f *LocalTemplater) validateGeneratedFiles(
	fs afero.Afero,
	dir string,
) error {
	debug := level.Debug(log.With(f.Logger, "method", "validateGeneratedFiles"))

	debug.Log("event", "readdir", "folder", dir)
	files, err := fs.ReadDir(dir)
	if err != nil {
		debug.Log("event", "readdir.fail", "folder", dir)
		return errors.Wrapf(err, "failed to read folder %s", dir)
	}

	for _, file := range files {
		thisPath := filepath.Join(dir, file.Name())
		if file.IsDir() {
			err := f.validateGeneratedFiles(fs, thisPath)
			if err != nil {
				return err
			}
		} else {
			err := fixFile(fs, thisPath, file.Mode())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func fixFile(fs afero.Afero, thisPath string, mode os.FileMode) error {
	contents, err := fs.ReadFile(thisPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", thisPath)
	}

	scanner := bufio.NewScanner(bytes.NewReader(contents))

	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrapf(err, "failed to read lines from file %s", thisPath)
	}

	lines = fixLines(lines)

	var outputFile bytes.Buffer
	for idx, line := range lines {
		if idx+1 != len(lines) || contents[len(contents)-1] == '\n' {
			fmt.Fprintln(&outputFile, line)
		} else {
			// avoid adding trailing newlines
			fmt.Fprintf(&outputFile, line)
		}
	}

	err = fs.WriteFile(thisPath, outputFile.Bytes(), mode)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s after fixup", thisPath)
	}

	return nil
}

// applies all fixes to all lines provided
func fixLines(lines []string) []string {
	for idx, line := range lines {
		if arrayLineRegex.MatchString(line) {
			// line has `key:` and nothing else but whitespace
			if !checkIsChild(line, nextLine(idx, lines)) {
				// next line is not a child, so this key has no contents, add an empty array
				lines[idx] = line + " []"
			}
		} else if envLineRegex.MatchString(line) {
			// line has `env:` and nothing else but whitespace
			if !checkIsChild(line, nextLine(idx, lines)) {
				// next line is not a child, so env has no contents, add an empty object
				lines[idx] = line + " []"
			}
		} else if valueLineRegex.MatchString(line) {
			// line has `value:` and nothing else but whitespace
			if !checkIsChild(line, nextLine(idx, lines)) {
				// next line is not a child, so value has no contents, add an empty string
				lines[idx] = line + ` ""`
			}
		} else if nullValueLineRegex.MatchString(line) {
			// line has `value: null`
			matches := nullValueLineRegex.FindStringSubmatch(line)

			if len(matches) >= 2 && matches[0] == line {
				lines[idx] = matches[1] + ` ""`
			}
		}
	}

	return lines
}

// returns true if the second line is a child of the first
func checkIsChild(firstLine, secondLine string) bool {
	cutset := fmt.Sprintf(" \t")
	firstIndentation := len(firstLine) - len(strings.TrimLeft(firstLine, cutset))
	secondIndentation := len(secondLine) - len(strings.TrimLeft(secondLine, cutset))

	if firstIndentation < secondIndentation {
		// if the next line is more indented, it's a child
		return true
	}

	if firstIndentation == secondIndentation {
		if secondLine[secondIndentation] == '-' {
			// if the next line starts with '-' and is on the same indentation, it's a child
			return true
		}
	}

	return false
}

// returns the next line after idx that is not entirely whitespace or a comment. If there are no lines meeting these criteria, returns ""
func nextLine(idx int, lines []string) string {
	if idx+1 >= len(lines) {
		return ""
	}

	if len(strings.TrimSpace(lines[idx+1])) > 0 {
		if strings.TrimSpace(lines[idx+1])[0] != '#' {
			return lines[idx+1]
		}
	}

	return nextLine(idx+1, lines)
}
