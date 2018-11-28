package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/replicatedhq/ship/integration"
	"github.com/replicatedhq/ship/pkg/cli"
	"gopkg.in/yaml.v2"
)

type TestMetadata struct {
	CustomerID          string `yaml:"customer_id"`
	InstallationID      string `yaml:"installation_id"`
	ReleaseVersion      string `yaml:"release_version"`
	SetChannelName      string `yaml:"set_channel_name"`
	SetGitHubContents   string `yaml:"set_github_contents"`
	DisableOnline       bool   `yaml:"disable_online"`
	NoStateFile         bool   `yaml:"no_state_file"` // used to denote that there is no input state.json
	SetEntitlementsJSON string `yaml:"set_entitlements_json"`
	//debugging
	SkipCleanup bool `yaml:"skip_cleanup"`
}

func TestShipApp(t *testing.T) {
	RegisterFailHandler(Fail)
	format.MaxDepth = 30
	RunSpecs(t, "ship app")
}

var _ = Describe("ship app", func() {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	dockerClient.NegotiateAPIVersion(context.Background())

	integrationDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	files, err := ioutil.ReadDir(integrationDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() {
			Context(fmt.Sprintf("When the spec in %q is run", file.Name()), func() {
				testPath := path.Join(integrationDir, file.Name())
				testInputPath := path.Join(testPath, "input")
				var testOutputPath string
				var testMetadata TestMetadata

				BeforeEach(func() {
					os.Setenv("NO_OS_EXIT", "1")
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the test metadata
					testMetadata = readMetadata(testPath)
				}, 20)

				AfterEach(func() {
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						// remove the temporary directory
						err := os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}
					os.Chdir(integrationDir)
				}, 20)

				It("Should output files matching those expected when running app command in local mode", func() {
					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					upstream := fmt.Sprintf(
						"%s?customer_id=%s&installation_id=%s&release_semver=%s",
						path.Join(testInputPath, ".ship/ship.yml"),
						testMetadata.CustomerID, testMetadata.InstallationID, testMetadata.ReleaseVersion,
					)
					args := []string{
						"init",
						upstream,
						fmt.Sprintf("--set-channel-name=%s", testMetadata.SetChannelName),
						fmt.Sprintf("--set-github-contents=%s", testMetadata.SetGitHubContents),
						"--headless",
						"--log-level=off",
						"--terraform-apply-yes",
					}
					if !testMetadata.NoStateFile {
						args = append(args, fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")))
					}

					if testMetadata.SetEntitlementsJSON != "" {
						args = append(args, fmt.Sprintf("--set-entitlements-json=%s", testMetadata.SetEntitlementsJSON))
					}

					cmd.SetArgs(args)
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					//compare the files in the temporary directory with those in the "expected" directory
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, map[string]string{
						"__upstream__": strings.Replace(upstream, "&", "\\u0026", -1),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())
				}, 60)

				It("Should output files matching those expected when communicating with the graphql api", func() {
					if testMetadata.DisableOnline {
						Skip("Online test skipped")
					}
					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					upstream := fmt.Sprintf(
						"staging.replicated.app/integration?customer_id=%s&installation_id=%s&release_semver=%s",
						testMetadata.CustomerID, testMetadata.InstallationID, testMetadata.ReleaseVersion,
					)
					args := []string{
						"init",
						upstream,
						"--headless",
						"--log-level=off",
						"--terraform-apply-yes",
					}
					if !testMetadata.NoStateFile {
						args = append(args, fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")))
					}
					cmd.SetArgs(args)
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					//compare the files in the temporary directory with those in the "expected" directory
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, map[string]string{
						"__upstream__": strings.Replace(upstream, "&", "\\u0026", -1),
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())
				}, 60)
			})
		}
	}
})

func readMetadata(testPath string) TestMetadata {
	var testMetadata TestMetadata
	metadataBytes, err := ioutil.ReadFile(path.Join(testPath, "metadata.yaml"))
	Expect(err).NotTo(HaveOccurred())
	err = yaml.Unmarshal(metadataBytes, &testMetadata)
	Expect(err).NotTo(HaveOccurred())

	return testMetadata
}
