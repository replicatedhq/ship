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
	"github.com/onsi/gomega/format"
	"github.com/replicatedhq/ship/integration"
	"github.com/replicatedhq/ship/pkg/cli"
	"gopkg.in/yaml.v2"
)

type TestMetadata struct {
	Upstream     string                `yaml:"upstream"`
	Fork         string                `yaml:"fork"`
	Args         []string              `yaml:"args"`
	IgnoredKeys  []map[string][]string `yaml:"ignoredKeys"`
	IgnoredFiles []string              `yaml:"ignoredFiles"`
}

func TestUnfork(t *testing.T) {
	RegisterFailHandler(Fail)
	format.MaxDepth = 30
	RunSpecs(t, "ship unfork")
}

var _ = Describe("ship unfork", func() {
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
					os.Setenv("NO_OS_EXIT", "1")
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the test metadata
					testMetadata = readMetadata(testPath)

				}, 20)

				AfterEach(func() {
					os.Chdir(integrationDir)
				}, 20)

				It("Should output the expected files", func() {
					replacements := map[string]string{}

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"unfork",
						"--upstream",
						testMetadata.Upstream,
						testMetadata.Fork,
						"--log-level=off",
					}, testMetadata.Args...))
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					// compare the files in the temporary directory with those in the "expected" directory
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, replacements, testMetadata.IgnoredFiles, testMetadata.IgnoredKeys)
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

func readMetadata(testPath string) TestMetadata {
	var testMetadata TestMetadata
	metadataBytes, err := ioutil.ReadFile(path.Join(testPath, "metadata.yaml"))
	Expect(err).NotTo(HaveOccurred())
	err = yaml.Unmarshal(metadataBytes, &testMetadata)
	Expect(err).NotTo(HaveOccurred())

	return testMetadata
}
