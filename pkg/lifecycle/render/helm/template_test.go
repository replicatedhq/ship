package helm

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/emosbaugh/yaml"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/process"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/test-mocks/helm"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/chartutil"
)

func TestLocalTemplater(t *testing.T) {
	tests := []struct {
		name                string
		describe            string
		expectError         string
		helmOpts            []string
		helmValues          map[string]interface{}
		expectedHelmValues  []string
		templateContext     map[string]interface{}
		channelName         string
		expectedChannelName string
		ontemplate          func(req *require.Assertions, mockFs afero.Afero) func(chartRoot string, args []string) error
		state               *state2.State
		requirements        *chartutil.Requirements
		repoAdd             []string
		namespace           string
	}{
		{
			name:        "helm test proper args",
			describe:    "test that helm is invoked with the proper args. The subprocess will fail if its not called with the args set in EXPECT_HELM_ARGV",
			expectError: "",
		},
		{
			name:        "helm with set value",
			describe:    "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			expectError: "",
			helmOpts:    []string{"--set", "service.clusterIP=10.3.9.2"},
		},
		{
			name:        "helm with subcharts",
			describe:    "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			expectError: "",
			helmOpts:    []string{"--set", "service.clusterIP=10.3.9.2"},
			ontemplate: func(req *require.Assertions, mockFs afero.Afero) func(chartRoot string, args []string) error {
				return func(chartRoot string, args []string) error {
					mockFolderPathToCreate := path.Join(constants.ShipPathInternalTmp, "chartrendered", "frobnitz", "templates")
					req.NoError(mockFs.MkdirAll(mockFolderPathToCreate, 0755))
					mockChartsPathToCreate := path.Join(constants.ShipPathInternalTmp, "chartrendered", "frobnitz", "charts")
					req.NoError(mockFs.MkdirAll(mockChartsPathToCreate, 0755))
					return nil
				}
			},
		},
		{
			name:        "helm values from asset value",
			describe:    "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
			expectedHelmValues: []string{
				"--set", "service.clusterIP=10.3.9.2",
			},
		},
		{
			name:        "helm replaces spacial characters in ",
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
			expectedHelmValues: []string{
				"--set", "service.clusterIP=10.3.9.2",
			},
			channelName:         "1-2-3---------frobnitz",
			expectedChannelName: "1-2-3---------frobnitz",
		},
		{
			name:        "helm templates values from context",
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "{{repl ConfigOption \"cluster_ip\"}}",
			},
			templateContext: map[string]interface{}{
				"cluster_ip": "10.3.9.2",
			},
			expectedHelmValues: []string{
				"--set", "service.clusterIP=10.3.9.2",
			},
			channelName:         "1-2-3---------frobnitz",
			expectedChannelName: "1-2-3---------frobnitz",
		},
		{
			name:        "helm values from asset value with incubator requirement",
			describe:    "calls helm repo add",
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
			expectedHelmValues: []string{
				"--set", "service.clusterIP=10.3.9.2",
			},
			requirements: &chartutil.Requirements{
				Dependencies: []*chartutil.Dependency{
					{
						Repository: "https://kubernetes-charts-incubator.storage.googleapis.com/",
					},
				},
			},
			repoAdd: []string{"kubernetes-charts-incubator", "https://kubernetes-charts-incubator.storage.googleapis.com/"},
		},
		{
			name:        "helm template with namespace in state",
			describe:    "template uses namespace from state",
			expectError: "",
			state: &state2.State{
				V1: &state2.V1{
					Namespace: "test-namespace",
				},
			},
			namespace: "test-namespace",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			mockState := state.NewMockManager(mc)
			mockCommands := helm.NewMockCommands(mc)
			memMapFs := afero.MemMapFs{}
			mockFs := afero.Afero{Fs: &memMapFs}
			tpl := &LocalTemplater{
				Commands:       mockCommands,
				Logger:         testLogger,
				FS:             mockFs,
				BuilderBuilder: templates.NewBuilderBuilder(testLogger, viper.New(), &state.MockManager{}),
				Viper:          viper.New(),
				StateManager:   mockState,
				process:        process.Process{Logger: testLogger},
			}

			channelName := "frobnitz"
			expectedChannelName := "frobnitz"
			if test.channelName != "" {
				channelName = test.channelName
			}
			if test.expectedChannelName != "" {
				expectedChannelName = test.expectedChannelName
			}

			if test.templateContext == nil {
				test.templateContext = map[string]interface{}{}
			}

			if test.state == nil {
				mockState.EXPECT().CachedState().Return(state2.State{
					V1: &state2.V1{
						HelmValues:  "we fake",
						ReleaseName: channelName,
					},
				}, nil)
			} else {
				testState := *test.state
				testState.V1.ReleaseName = channelName
				mockState.EXPECT().CachedState().Return(testState, nil)
			}

			chartRoot := "/tmp/chartroot"
			optionAndValuesArgs := append(
				test.helmOpts,
				test.expectedHelmValues...,
			)

			if test.requirements != nil {
				requirementsB, err := yaml.Marshal(test.requirements)
				req.NoError(err)
				err = mockFs.WriteFile(path.Join(chartRoot, "requirements.yaml"), requirementsB, 0755)
				req.NoError(err)
			}

			templateArgs := append(
				[]string{
					"--output-dir", ".ship/tmp/chartrendered",
					"--name", expectedChannelName,
				},
				optionAndValuesArgs...,
			)

			if len(test.namespace) > 0 {
				templateArgs = addArgIfNotPresent(templateArgs, "--namespace", test.namespace)
			} else {
				templateArgs = addArgIfNotPresent(templateArgs, "--namespace", "default")
			}

			mockCommands.EXPECT().Init().Return(nil)
			if test.requirements != nil {
				absTempHelmHome, err := filepath.Abs(constants.InternalTempHelmHome)
				req.NoError(err)
				mockCommands.EXPECT().RepoAdd(test.repoAdd[0], test.repoAdd[1], absTempHelmHome)

				requirementsB, err := mockFs.ReadFile(filepath.Join(chartRoot, "requirements.yaml"))
				req.NoError(err)
				chartRequirements := chartutil.Requirements{}
				err = yaml.Unmarshal(requirementsB, &chartRequirements)
				req.NoError(err)

				mockCommands.EXPECT().MaybeDependencyUpdate(chartRoot, chartRequirements).Return(nil)
			} else {
				mockCommands.EXPECT().MaybeDependencyUpdate(chartRoot, chartutil.Requirements{}).Return(nil)
			}

			if test.ontemplate != nil {
				mockCommands.EXPECT().Template(chartRoot, templateArgs).DoAndReturn(test.ontemplate(req, mockFs))
			} else {

				mockCommands.EXPECT().Template(chartRoot, templateArgs).DoAndReturn(func(rootDir string, args []string) error {
					mockFolderPathToCreate := path.Join(constants.ShipPathInternalTmp, "chartrendered", expectedChannelName, "templates")
					req.NoError(mockFs.MkdirAll(mockFolderPathToCreate, 0755))
					return nil
				})
			}

			err := tpl.Template(
				"/tmp/chartroot",
				root.Fs{
					Afero:    mockFs,
					RootPath: "",
				},
				api.HelmAsset{
					AssetShared: api.AssetShared{
						Dest: "k8s/",
					},
					HelmOpts: test.helmOpts,
					Values:   test.helmValues,
				},
				api.ReleaseMetadata{
					Semver:      "1.0.0",
					ChannelName: channelName,
					ShipAppMetadata: api.ShipAppMetadata{
						Name: expectedChannelName,
					},
				},
				[]libyaml.ConfigGroup{},
				test.templateContext,
			)

			t.Logf("checking error %v", err)
			if test.expectError == "" {
				req.NoError(err)
			} else {
				req.Error(err, "expected error "+test.expectError)
				req.Equal(test.expectError, err.Error())
			}

		})
	}
}

func TestTryRemoveKustomizeBasePath(t *testing.T) {
	tests := []struct {
		name        string
		describe    string
		baseDir     string
		expectError bool
	}{
		{
			name:        "base exists",
			describe:    "ensure base is removed and removeAll doesn't error",
			baseDir:     constants.KustomizeBasePath,
			expectError: false,
		},
		{
			name:        "base does not exist",
			describe:    "missing base, ensure removeAll doesn't error",
			baseDir:     "",
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}

			// create the base directory
			err := fakeFS.MkdirAll(path.Join(test.baseDir, "myCoolManifest.yaml"), 0777)
			req.NoError(err)

			// verify path actually exists
			successfulMkdirAll, err := fakeFS.DirExists(path.Join(test.baseDir, "myCoolManifest.yaml"))
			req.True(successfulMkdirAll)
			req.NoError(err)

			ft := &LocalTemplater{
				FS:     fakeFS,
				Logger: testLogger,
			}

			removeErr := ft.FS.RemoveAll(constants.KustomizeBasePath)

			if test.expectError {
				req.Error(removeErr)
			} else {
				if dirExists, existErr := ft.FS.DirExists(constants.KustomizeBasePath); dirExists {
					req.NoError(existErr)
					// if dir exists, we expect tryRemoveKustomizeBasePath to have err'd
					req.Error(removeErr)
				} else {
					// if dir does not exist, we expect tryRemoveKustomizeBasePath to have succeeded without err'ing
					req.NoError(removeErr)
				}
			}
		})
	}
}

func Test_addArgIfNotPresent(t *testing.T) {
	type args struct {
		existingArgs []string
		newArg       string
		newDefault   string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty",
			args: args{
				existingArgs: []string{},
				newArg:       "--test",
				newDefault:   "newDefault",
			},
			want: []string{"--test", "newDefault"},
		},
		{
			name: "not present, not empty",
			args: args{
				existingArgs: []string{"--notTest", "notDefault"},
				newArg:       "--test",
				newDefault:   "newDefault",
			},
			want: []string{"--notTest", "notDefault", "--test", "newDefault"},
		},
		{
			name: "present",
			args: args{
				existingArgs: []string{"--test", "notDefault"},
				newArg:       "--test",
				newDefault:   "newDefault",
			},
			want: []string{"--test", "notDefault"},
		},
		{
			name: "present with others",
			args: args{
				existingArgs: []string{"--notTest", "notDefault", "--test", "alsoNotDefault"},
				newArg:       "--test",
				newDefault:   "newDefault",
			},
			want: []string{"--notTest", "notDefault", "--test", "alsoNotDefault"},
		},
		{
			name: "present as substring",
			args: args{
				existingArgs: []string{"--notTest", "notDefault", "abc--test", "alsoNotDefault"},
				newArg:       "--test",
				newDefault:   "newDefault",
			},
			want: []string{"--notTest", "notDefault", "abc--test", "alsoNotDefault", "--test", "newDefault"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got := addArgIfNotPresent(tt.args.existingArgs, tt.args.newArg, tt.args.newDefault)

			req.Equal(tt.want, got)
		})
	}
}

func Test_validateGeneratedFiles(t *testing.T) {

	type file struct {
		contents string
		path     string
	}
	tests := []struct {
		name        string
		inputFiles  []file
		dir         string
		outputFiles []file
	}{
		{
			name:        "no_files",
			dir:         "",
			inputFiles:  []file{},
			outputFiles: []file{},
		},
		{
			name: "irrelevant_files",
			dir:  "test",
			inputFiles: []file{
				{
					path:     "outside",
					contents: `irrelevant`,
				},
				{
					path: "test/inside",
					contents: `irrelevant
`,
				},
			},
			outputFiles: []file{
				{
					path:     "outside",
					contents: `irrelevant`,
				},
				{
					path: "test/inside",
					contents: `irrelevant
`,
				},
			},
		},
		{
			name: "relevant_args_files",
			dir:  "test",
			inputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  args: {}`,
				},
				{
					path:     "test/missingArgs.yaml",
					contents: `  args:`,
				},
				{
					path: "test/notMissingMultilineArgs.yaml",
					contents: `
  args:
    something
  args:
  - something`,
				},
				{
					path: "test/missingMultilineArgs.yaml",
					contents: `
  args:
  something:`,
				},
			},
			outputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  args: {}`,
				},
				{
					path:     "test/missingArgs.yaml",
					contents: `  args: []`,
				},
				{
					path: "test/notMissingMultilineArgs.yaml",
					contents: `
  args:
    something
  args:
  - something`,
				},
				{
					path: "test/missingMultilineArgs.yaml",
					contents: `
  args: []
  something:`,
				},
			},
		},
		{
			name: "relevant_env_files",
			dir:  "test",
			inputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  env: []`,
				},
				{
					path:     "test/missingEnv.yaml",
					contents: `  env:`,
				},
				{
					path: "test/notMissingMultilineEnv.yaml",
					contents: `
  env:
    something
  env:
  - something`,
				},
				{
					path: "test/missingMultilineEnv.yaml",
					contents: `
  env:
  something:`,
				},
			},
			outputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  env: []`,
				},
				{
					path:     "test/missingEnv.yaml",
					contents: `  env: []`,
				},
				{
					path: "test/notMissingMultilineEnv.yaml",
					contents: `
  env:
    something
  env:
  - something`,
				},
				{
					path: "test/missingMultilineEnv.yaml",
					contents: `
  env: []
  something:`,
				},
			},
		},
		{
			name: "relevant_value_files",
			dir:  "test",
			inputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  value: {}`,
				},
				{
					path:     "test/missingValue.yaml",
					contents: `  value:`,
				},
			},
			outputFiles: []file{
				{
					path:     "test/something.yaml",
					contents: `  value: {}`,
				},
				{
					path:     "test/missingValue.yaml",
					contents: `  value: ""`,
				},
			},
		},
		{
			name: "blank lines",
			dir:  "test",
			inputFiles: []file{
				{
					path: "test/blank_line_env.yaml",
					contents: `
  env:

    item
`,
				},
				{
					path: "test/blank_line_args.yaml",
					contents: `
  args:

    item
`,
				},
			},
			outputFiles: []file{
				{
					path: "test/blank_line_env.yaml",
					contents: `
  env:

    item
`,
				},
				{
					path: "test/blank_line_args.yaml",
					contents: `
  args:

    item
`,
				},
			},
		},
		{
			name: "comment lines",
			dir:  "test",
			inputFiles: []file{
				{
					path: "test/comment_line_env.yaml",
					contents: `
  env:
    #item

  env:
  #item
    item2
`,
				},
				{
					path: "test/comment_line_args.yaml",
					contents: `
  args:
    #item

  args:
  #item
    item2
`,
				},
			},
			outputFiles: []file{
				{
					path: "test/comment_line_env.yaml",
					contents: `
  env: []
    #item

  env:
  #item
    item2
`,
				},
				{
					path: "test/comment_line_args.yaml",
					contents: `
  args: []
    #item

  args:
  #item
    item2
`,
				},
			},
		},
		{
			name: "null values",
			dir:  "test",
			inputFiles: []file{
				{
					path: "test/null_values.yaml",
					contents: `
  value: null
    #item

  value:
    null

  value:
    value: null
`,
				},
			},
			outputFiles: []file{
				{
					path: "test/null_values.yaml",
					contents: `
  value: ""
    #item

  value:
    null

  value:
    value: ""
`,
				},
			},
		},
		{
			name: "templated values",
			dir:  "test",
			inputFiles: []file{
				{
					path: "test/null_values.yaml",
					contents: `
  value:

{{ template }}

  value:
  {{ template }}

  value:
    value: {{ template }}
`,
				},
			},
			outputFiles: []file{
				{
					path: "test/null_values.yaml",
					contents: `
  value:

{{ template }}

  value:
  {{ template }}

  value:
    value: {{ template }}
`,
				},
			},
		},
		{
			name: "everything",
			dir:  "test",
			inputFiles: []file{
				{
					path: "test/everything.yaml",
					contents: `
  args:
  env:
  volumes:
  value:
  value: null
  initContainers:
`,
				},
			},
			outputFiles: []file{
				{
					path: "test/everything.yaml",
					contents: `
  args: []
  env: []
  volumes: []
  value: ""
  value: ""
  initContainers: []
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}

			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}
			lt := &LocalTemplater{
				FS:     fakeFS,
				Logger: testLogger,
			}

			// add inputFiles to fakeFS
			for _, file := range tt.inputFiles {
				req.NoError(fakeFS.WriteFile(file.path, []byte(file.contents), os.FileMode(777)))
			}

			req.NoError(lt.validateGeneratedFiles(fakeFS, tt.dir))

			// check outputFiles from fakeFS
			for _, file := range tt.outputFiles {
				contents, err := fakeFS.ReadFile(file.path)
				req.NoError(err)
				req.Equal(file.contents, string(contents), "expected %s contents to be equal", file.path)
			}
		})
	}
}

func TestLocalTemplater_writeStateHelmValuesTo(t *testing.T) {
	tests := []struct {
		name                 string
		dest                 string
		defaultValuesPath    string
		defaultValuesContent string
	}{
		{
			name:              "simple",
			dest:              "some/values.yaml",
			defaultValuesPath: "random/values.yaml",
			defaultValuesContent: `
something: maybe
`,
		},
	}
	for _, tt := range tests {
		req := require.New(t)
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			mockState := state.NewMockManager(mc)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			err := mockFs.WriteFile(tt.defaultValuesPath, []byte(tt.defaultValuesContent), 0755)
			req.NoError(err)

			mockState.EXPECT().CachedState().Return(state2.State{V1: &state2.V1{}}, nil)
			f := &LocalTemplater{
				Logger:       &logger.TestLogger{T: t},
				FS:           mockFs,
				StateManager: mockState,
			}
			err = f.writeStateHelmValuesTo(tt.dest, tt.defaultValuesPath)
			req.NoError(err)

			readFileB, err := mockFs.ReadFile(tt.dest)
			req.NoError(err)
			req.Equal(tt.defaultValuesContent, string(readFileB))
		})
	}
}
