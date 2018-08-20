package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/integration"
	"github.com/replicatedhq/ship/pkg/cli"
	"github.com/replicatedhq/ship/pkg/e2e"
	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type TestMetadata struct {
	CustomerID     string `yaml:"customer_id"`
	InstallationID string `yaml:"installation_id"`
	ReleaseVersion string `yaml:"release_version"`
	SetChannelName string `yaml:"set_channel_name"`
	Flavor         string `yaml:"flavor"`
	DisableOnline  bool   `yaml:"disable_online"`

	// debugging
	SkipCleanup bool `yaml:"skip_cleanup"`
}

func TestInitReplicatedApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ship init replicated.app")
}

var _ = Describe("ship init replicated.app/...", func() {
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
			When(fmt.Sprintf("the spec in %q is run", file.Name()), func() {
				testPath := path.Join(integrationDir, file.Name())
				testInputPath := path.Join(testPath, "input")
				var testOutputPath string
				var testMetadata TestMetadata
				var installationID string
				customerEndpoint := os.Getenv("SHIP_INTEGRATION_CUSTOMER_ENDPOINT")
				vendorEndpoint := os.Getenv("SHIP_INTEGRATION_VENDOR_ENDPOINT")
				vendorToken := os.Getenv("SHIP_INTEGRATION_VENDOR_TOKEN")
				if customerEndpoint == "" {
					customerEndpoint = "https://pg.staging.replicated.com/graphql"
				}
				if vendorEndpoint == "" {
					vendorEndpoint = "https://g.staging.replicated.com/graphql"
				}

				BeforeEach(func(done chan<- interface{}) {
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the test metadata
					testMetadata = readMetadata(testPath)

					// if a token is provided, try to ensure the release matches what we have here in the repo

					if vendorToken != "" {
						channelName := fmt.Sprintf("integration replicated.app %s", filepath.Base(testPath))
						installationID = createRelease(vendorEndpoint, vendorToken, testInputPath, testMetadata, channelName)
					}
					close(done)

				}, 20)

				AfterEach(func() {
					if !testMetadata.SkipCleanup {
						// remove the temporary directory
						err := os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}
					os.Chdir(integrationDir)
				}, 20)

				It("Should output files matching those expected when communicating with the graphql api", func() {
					if testMetadata.DisableOnline {
						Skip("Online test skipped")
					}

					isStaging := strings.Contains(customerEndpoint, "staging")
					upstream := "replicated.app/some-cool-ci-tool"
					if isStaging {
						upstream = "staging.replicated.app/some-cool-ci-tool"
					}

					// this should probably be url encoded but whatever
					upstream = fmt.Sprintf(
						"%s?installation_id=%s&customer_id=%s",
						upstream,
						installationID,
						testMetadata.CustomerID,
					)

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"init",
						upstream,
						"--headless",
						fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")),
						"--log-level=off",
					}))
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					// compare the files in the temporary directory with those in the "expected" directory
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())
				}, 60)
			})
		}
	}
})

func createRelease(
	vendorEndpoint string,
	vendorToken string,
	testInputPath string,
	testMetadata TestMetadata,
	channelName string,
) string {
	endpointURL, err := url.Parse(vendorEndpoint)
	Expect(err).NotTo(HaveOccurred())
	vendorClient := &e2e.GraphQLClient{
		GQLServer: endpointURL,
		Token:     vendorToken,
		Logger: logger.New(
			viper.GetViper(),
			afero.Afero{Fs: afero.NewMemMapFs()},
		),
	}
	releaseContents, err := ioutil.ReadFile(path.Join(testInputPath, ".ship/release.yml"))
	Expect(err).NotTo(HaveOccurred())
	channel, err := vendorClient.GetOrCreateChannel(channelName)
	Expect(err).NotTo(HaveOccurred())
	_, err = vendorClient.PromoteRelease(
		string(releaseContents),
		channel.ID,
		testMetadata.ReleaseVersion,
		fmt.Sprintf("integration tests running on %s", time.Now()),
	)
	Expect(err).NotTo(HaveOccurred())
	installationID, err := vendorClient.EnsureCustomerOnChannel(testMetadata.CustomerID, channel.ID)
	Expect(err).NotTo(HaveOccurred())
	return installationID
}

func readMetadata(testPath string) TestMetadata {
	var testMetadata TestMetadata
	metadataBytes, err := ioutil.ReadFile(path.Join(testPath, "metadata.yaml"))
	Expect(err).NotTo(HaveOccurred())
	err = yaml.Unmarshal(metadataBytes, &testMetadata)
	Expect(err).NotTo(HaveOccurred())

	return testMetadata
}
