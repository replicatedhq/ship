package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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
	Upstream     string   `yaml:"upstream"`
	Args         []string `yaml:"args"`
	MakeAbsolute bool     `yaml:"make_absolute"`
	// debugging
	SkipCleanup bool   `yaml:"skip_cleanup"`
	ValuesFile  string `yaml:"valuesFile"`
}

func TestInit(t *testing.T) {
	RegisterFailHandler(Fail)
	format.MaxDepth = 30
	RunSpecs(t, "ship init")
}

var _ = Describe("ship init with arbitrary upstream", func() {
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
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						// remove the temporary directory
						err := os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}
					os.Chdir(integrationDir)
				}, 20)

				It("Should output the expected files", func() {
					replacements := map[string]string{}
					absoluteUpstream := testMetadata.Upstream

					if testMetadata.MakeAbsolute {
						relativePath := testMetadata.Upstream
						pwdRoot, err := os.Getwd()
						Expect(err).NotTo(HaveOccurred())
						pwdRoot, err = filepath.Abs(pwdRoot)
						Expect(err).NotTo(HaveOccurred())
						absolutePath := filepath.Join(pwdRoot, "..")
						absoluteUpstream = fmt.Sprintf("file::%s", filepath.Join(absolutePath, relativePath))
						replacements["__upstream__"] = absoluteUpstream
					}

					if testMetadata.ValuesFile != "" {
						relativePath := testMetadata.ValuesFile
						absolutePath, err := filepath.Abs(path.Join(testPath, relativePath))
						Expect(err).NotTo(HaveOccurred())
						Expect(err).NotTo(HaveOccurred())
						testMetadata.Args = append(testMetadata.Args, fmt.Sprintf("--helm-values-file=%s", absolutePath))
					}

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"init",
						absoluteUpstream,
						"--headless",
						"--log-level=off",
					}, testMetadata.Args...))

					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

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

func readMetadata(testPath string) TestMetadata {
	var testMetadata TestMetadata
	metadataBytes, err := ioutil.ReadFile(path.Join(testPath, "metadata.yaml"))
	Expect(err).NotTo(HaveOccurred())
	err = yaml.Unmarshal(metadataBytes, &testMetadata)
	Expect(err).NotTo(HaveOccurred())

	return testMetadata
}
