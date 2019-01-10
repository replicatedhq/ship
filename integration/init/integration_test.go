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

					preserveState := argsContains(testMetadata.Args, "--preserve-state")
					if preserveState {
						moveInputStateJson(testPath, testOutputPath)
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

					watchTestMetadataArgs := argsFilter(testMetadata.Args, func(arg string) bool {
						return arg != "--preserve-state"
					})

					// run 'ship watch' and expect no error to occur
					watchCmd := cli.RootCmd()
					watchBuf := new(bytes.Buffer)
					watchCmd.SetOutput(watchBuf)
					watchCmd.SetArgs(append([]string{"watch", "--exit"}, watchTestMetadataArgs...))
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

func moveInputStateJson(testPath, testOutputPath string) {
	stateInputAbsolutePath, err := filepath.Abs(path.Join(testPath, "input", "state.json"))
	Expect(err).NotTo(HaveOccurred())

	stateInput, err := ioutil.ReadFile(stateInputAbsolutePath)
	Expect(err).NotTo(HaveOccurred())

	outputAbsolutePath, err := filepath.Abs(path.Join(testOutputPath, ".ship", "state.json"))
	Expect(err).NotTo(HaveOccurred())

	err = os.Mkdir(filepath.Dir(outputAbsolutePath), 0777)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(outputAbsolutePath, stateInput, 0777)
	Expect(err).NotTo(HaveOccurred())
}

func argsContains(args []string, containArg string) bool {
	for _, arg := range args {
		if arg == containArg {
			return true
		}
	}
	return false
}

func argsFilter(args []string, argPredicate func(arg string) bool) []string {
	filteredArgs := []string{}
	for _, arg := range args {
		if argPredicate(arg) {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	return filteredArgs
}
