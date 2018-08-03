package helm

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"reflect"
	"strings"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestForkTemplater(t *testing.T) {
	tests := []struct {
		name            string
		describe        string
		helmForkEnv     []string
		expectError     string
		helmOpts        []string
		helmValues      map[string]interface{}
		templateContext map[string]interface{}
		channelName     string
	}{
		{
			name:     "helm crashes",
			describe: "ensure that we bubble up an informative error if the forked process crashes",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"CRASHING_HELM_ERROR=I am helm and I crashed",
			},
			expectError: `execute helm: exit status 1: stdout: "I am helm and I crashed"; stderr: "";`,
		},
		{
			//
			name:     "helm bad args",
			describe: "this is more of a negative test of our exec-mocking framework -- to make sure that we can properly validate that proper args were passed",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				// this is janky, but works for our purposes, use pipe | for separator, since its unlikely to be in argv
				"EXPECT_HELM_ARGV=--foo|bar|--output-dir|fake",
			},
			expectError: fmt.Sprintf(
				"execute helm: exit status 2: stdout: \"\"; stderr: \"expected args [--foo bar --output-dir fake], got args [template /tmp/chartroot --output-dir %s --name frobnitz]; FAIL\";",
				constants.RenderedHelmTempPath,
			),
		},
		{
			name:     "helm test proper args",
			describe: "test that helm is invoked with the proper args. The subprocess will fail if its not called with the args set in EXPECT_HELM_ARGV",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"EXPECT_HELM_ARGV=" +
					"template|" +
					"/tmp/chartroot|" +
					"--output-dir|" + constants.RenderedHelmTempPath + "|" +
					"--name|frobnitz",
			},
			expectError: "",
		},
		{
			name:     "helm with set value",
			describe: "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"EXPECT_HELM_ARGV=" +
					"template|" +
					"/tmp/chartroot|" +
					"--output-dir|" + constants.RenderedHelmTempPath + "|" +
					"--name|frobnitz|" +
					"--set|service.clusterIP=10.3.9.2",
			},
			expectError: "",
			helmOpts:    []string{"--set", "service.clusterIP=10.3.9.2"},
		},
		{
			name:     "helm values from asset value",
			describe: "ensure any helm.helm_opts are forwarded down to the call to `helm template`",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"EXPECT_HELM_ARGV=" +
					"template|" +
					"/tmp/chartroot|" +
					"--output-dir|" + constants.RenderedHelmTempPath + "|" +
					"--name|frobnitz|" +
					"--set|service.clusterIP=10.3.9.2",
			},
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
		},
		{
			name: "helm replaces spacial characters in ",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"EXPECT_HELM_ARGV=" +
					"template|" +
					"/tmp/chartroot|" +
					"--output-dir|" + constants.RenderedHelmTempPath + "|" +
					"--name|1-2-3---------frobnitz|" +
					"--set|service.clusterIP=10.3.9.2",
			},
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "10.3.9.2",
			},
			channelName: "1.2.3-$#(%*)@-frobnitz",
		},
		{
			name: "helm templates values from context",
			helmForkEnv: []string{
				"GOTEST_SUBPROCESS_MOCK=1",
				"EXPECT_HELM_ARGV=" +
					"template|" +
					"/tmp/chartroot|" +
					"--output-dir|" + constants.RenderedHelmTempPath + "|" +
					"--name|1-2-3---------frobnitz|" +
					"--set|service.clusterIP=10.3.9.2",
			},
			expectError: "",
			helmValues: map[string]interface{}{
				"service.clusterIP": "{{repl ConfigOption \"cluster_ip\"}}",
			},
			templateContext: map[string]interface{}{
				"cluster_ip": "10.3.9.2",
			},
			channelName: "1.2.3-$#(%*)@-frobnitz",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}
			tpl := &ForkTemplater{
				Helm: func() *exec.Cmd {
					cmd := exec.Command(os.Args[0], "-test.run=TestMockHelm")
					cmd.Env = append(os.Environ(), test.helmForkEnv...)
					return cmd
				},
				Logger:         testLogger,
				FS:             afero.Afero{Fs: afero.NewMemMapFs()},
				BuilderBuilder: templates.NewBuilderBuilder(testLogger),
			}

			channelName := "Frobnitz"
			if test.channelName != "" {
				channelName = test.channelName
			}

			if test.templateContext == nil {
				test.templateContext = map[string]interface{}{}
			}

			err := tpl.Template(
				"/tmp/chartroot",
				api.HelmAsset{
					AssetShared: api.AssetShared{
						Dest: "k8s/",
					},
					HelmOpts: test.helmOpts,
					Values:   test.helmValues,
				}, api.ReleaseMetadata{
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
