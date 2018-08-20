package helm

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/test-mocks/helm"

	"github.com/spf13/viper"

	"path"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/process"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
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
			channelName:         "1.2.3-$#(%*)@-frobnitz",
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
			channelName:         "1.2.3-$#(%*)@-frobnitz",
			expectedChannelName: "1-2-3---------frobnitz",
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
				BuilderBuilder: templates.NewBuilderBuilder(testLogger),
				Viper:          viper.New(),
				StateManager:   mockState,
				process:        process.Process{Logger: testLogger},
			}

			mockState.EXPECT().TryLoad().Return(state2.VersionedState{
				V1: &state2.V1{
					HelmValues: "we fake",
				},
			}, nil)

			channelName := "Frobnitz"
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

			chartRoot := "/tmp/chartroot"
			optionAndValuesArgs := append(
				test.helmOpts,
				test.expectedHelmValues...,
			)
			templateArgs := append(
				[]string{
					"--output-dir", constants.RenderedHelmTempPath,
					"--name", expectedChannelName,
				},
				optionAndValuesArgs...,
			)
			mockCommands.EXPECT().Init().Return(nil)
			mockCommands.EXPECT().DependencyUpdate(chartRoot).Return(nil)
			mockCommands.EXPECT().Template(chartRoot, templateArgs).Return(nil)

			mockFolderPathToCreate := path.Join(constants.RenderedHelmTempPath, expectedChannelName, "templates")
			req.NoError(mockFs.MkdirAll(mockFolderPathToCreate, 0755))

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
					HelmChartMetadata: api.HelmChartMetadata{
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

func TestTryRemoveRenderedHelmPath(t *testing.T) {
	tests := []struct {
		name        string
		describe    string
		baseDir     string
		expectError bool
	}{
		{
			name:        "base exists",
			describe:    "ensure base is removed and removeAll doesn't error",
			baseDir:     constants.RenderedHelmPath,
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

			removeErr := ft.FS.RemoveAll(constants.RenderedHelmPath)

			if test.expectError {
				req.Error(removeErr)
			} else {
				if dirExists, existErr := ft.FS.DirExists(constants.RenderedHelmPath); dirExists {
					req.NoError(existErr)
					// if dir exists, we expect tryRemoveRenderedHelmPath to have err'd
					req.Error(removeErr)
				} else {
					// if dir does not exist, we expect tryRemoveRenderedHelmPath to have succeeded without err'ing
					req.NoError(removeErr)
				}
			}
		})
	}
}
