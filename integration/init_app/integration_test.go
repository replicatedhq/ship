package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
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
	"github.com/replicatedhq/ship/pkg/e2e"
	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type TestMetadata struct {
	InstallationID string   `yaml:"installation_id"`
	CustomerID     string   `yaml:"customer_id"`
	ReleaseVersion string   `yaml:"release_version"`
	SetChannelName string   `yaml:"set_channel_name"`
	Flavor         string   `yaml:"flavor"`
	Args           []string `yaml:"args"`

	// debugging
	SkipCleanup bool `yaml:"skip_cleanup"`
}

func TestInitReplicatedApp(t *testing.T) {
	RegisterFailHandler(Fail)
	format.MaxDepth = 30
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
				var testOutputPath string
				var testMetadata TestMetadata

				BeforeEach(func(done chan<- interface{}) {
					os.Setenv("NO_OS_EXIT", "1")
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the test metadata
					testMetadata = readMetadata(testPath)

					// TODO - instead of getting installation ID, etc from test metadata create a release with the vendor api
					// TODO customer ID and vendor token will need to be read from environment variables
					// TODO so will the desired environment - staging vs prod

					close(done)
				}, 20)

				AfterEach(func() {
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						// remove the temporary directory
						err := os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}
					os.Chdir(integrationDir)
				}, 20)

				It("Should output files matching those expected when communicating with the graphql api", func() {

					upstream := "staging.replicated.app/some-cool-ci-tool"

					// this should probably be url encoded but whatever
					upstream = fmt.Sprintf(
						"%s?installation_id=%s&customer_id=%s",
						upstream,
						testMetadata.InstallationID,
						testMetadata.CustomerID,
					)

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"init",
						upstream,
						"--headless",
						"--log-level=off",
					}, testMetadata.Args...))
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					// these strings will be replaced in the "expected" yaml before comparison
					replacements := map[string]string{
						"__upstream__":       strings.Replace(upstream, "&", "\\u0026", -1), // this string is encoded within the output
						"__installationID__": testMetadata.InstallationID,
						"__customerID__":     testMetadata.CustomerID,
					}

					// compare the files in the temporary directory with those in the "expected" directory
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, replacements, []string{}, []map[string][]string{})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())

					// run 'ship watch' and expect no error to occur
					watchCmd := cli.RootCmd()
					watchBuf := new(bytes.Buffer)
					watchCmd.SetOutput(watchBuf)
					watchCmd.SetArgs(append([]string{"watch", "--exit"}, testMetadata.Args...))
					err = watchCmd.Execute()
					Expect(err).NotTo(HaveOccurred())
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
	customerID string,
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
		"integration tests",
	)
	Expect(err).NotTo(HaveOccurred())
	installationID, err := vendorClient.EnsureCustomerOnChannel(customerID, channel.ID)
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
