package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/integration"
	"github.com/replicatedhq/ship/pkg/cli"
	"gopkg.in/yaml.v2"
)

type TestMetadata struct {
	Chart   string `yaml:"chart"`
	Version string `yaml:"version"`
	RepoURL string `yaml:"repo_url"`

	// debugging
	SkipCleanup bool `yaml:"skip_cleanup"`
}

func TestInitReplicatedApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "integration")
}

var _ = Describe("ship init with 'helm fetch'", func() {
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

				BeforeEach(func() {
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the test metadata
					testMetadata = readMetadata(testPath)

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
					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"init",
						testMetadata.Chart,
						"--headless",
						"--log-level=off",
						"--chart-version",
						testMetadata.Version,
						"--chart-repo-url",
						testMetadata.RepoURL,
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

func readMetadata(testPath string) TestMetadata {
	var testMetadata TestMetadata
	metadataBytes, err := ioutil.ReadFile(path.Join(testPath, "metadata.yaml"))
	Expect(err).NotTo(HaveOccurred())
	err = yaml.Unmarshal(metadataBytes, &testMetadata)
	Expect(err).NotTo(HaveOccurred())

	return testMetadata
}
