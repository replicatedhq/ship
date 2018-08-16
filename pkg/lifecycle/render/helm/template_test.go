package helm

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/replicatedhq/ship/pkg/test-mocks/helm"

	"github.com/spf13/viper"

	"reflect"
	"strings"

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

func TestForkTemplater(t *testing.T) {
	tests := []struct {
		name                string
		describe            string
		helmForkEnv         []string
		expectError         string
		helmOpts            []string
		helmValues          map[string]interface{}
		expectedHelmValues  []string
		templateContext     map[string]interface{}
		channelName         string
		expectedChannelName string
	}{
		{
			name:     "helm test proper args",
			describe: "test that helm is invoked with the proper args. The subprocess will fail if its not called with the args set in EXPECT_HELM_ARGV",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
			},
			expectError: "",
		},
		{
			name:     "helm with set value",
			describe: "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
			},
			expectError: "",
			helmOpts:    []string{"--set", "service.clusterIP=10.3.9.2"},
		},
		{
			name:     "helm values from asset value",
			describe: "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
			},
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
			expectedHelmValues: []string{
				"--set", "service.clusterIP=10.3.9.2",
			},
		},
		{
			name: "helm replaces spacial characters in ",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
			},
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
			name: "helm templates values from context",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
			},
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
			tpl := &ForkTemplater{
				Helm: func() *exec.Cmd {
					cmd := exec.Command(os.Args[0], "-test.run=TestMockHelm")
					cmd.Env = append(os.Environ(), test.helmForkEnv...)
					return cmd
				},
				Commands:       mockCommands,
				Logger:         testLogger,
				FS:             afero.Afero{Fs: afero.NewMemMapFs()},
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
			fmt.Println("channelName", channelName)

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
			mockCommands.EXPECT().Template(chartRoot, templateArgs).Return(nil)

			err := tpl.Template(
				"/tmp/chartroot",
				root.Fs{
					Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
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

// thanks andrewG / hashifolks
func TestMockHelm(t *testing.T) {
	// this test does nothing when run normally, only when
	// invoked by other tests. Those tests should set this
	// env var in order to get the behavior
	if os.Getenv("GOTEST_SUBPROCESS_MOCK") == "" {
		return
	}

	receivedArgs := os.Args[2:]
	expectInit := []string{"init", "--client-only"}
	expectUpdate := []string{"dependency", "update", "/tmp/chartroot"}
	if reflect.DeepEqual(receivedArgs, expectInit) {
		// we good, these are exepcted calls, and we just need to test one type of forking
		os.Exit(0)
	}

	if reflect.DeepEqual(receivedArgs, expectUpdate) {
		// we good, these are exepcted calls
		os.Exit(0)
	}

	if os.Getenv("CRASHING_HELM_ERROR") != "" {
		fmt.Fprintf(os.Stdout, os.Getenv("CRASHING_HELM_ERROR"))
		os.Exit(1)
	}

	if os.Getenv("EXPECT_HELM_ARGV") != "" {
		// this is janky, but works for our purposes, use pipe | for separator, since its unlikely to be in argv
		expectedArgs := strings.Split(os.Getenv("EXPECT_HELM_ARGV"), "|")

		fmt.Fprintf(os.Stderr, "expected args %v, got args %v", expectedArgs, receivedArgs)
		if !reflect.DeepEqual(receivedArgs, expectedArgs) {
			fmt.Fprint(os.Stderr, "; FAIL")
			os.Exit(2)
		}

		os.Exit(0)
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

			ft := &ForkTemplater{
				FS:     fakeFS,
				Logger: testLogger,
			}

			removeErr := ft.tryRemoveRenderedHelmPath()

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
