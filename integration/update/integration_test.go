package integration

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
	"github.com/replicatedhq/ship/pkg/state"
	yaml "gopkg.in/yaml.v3"
)

type TestMetadata struct {
	Args []string `yaml:"args"`
	Skip bool     `yaml:"skip"`

	// debugging
	SkipCleanup  bool     `yaml:"skip_cleanup"`
	IgnoredFiles []string `yaml:"ignoredFiles"`
}

func TestShipUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	format.MaxDepth = 30
	RunSpecs(t, "ship update")
}

var _ = Describe("ship update", func() {
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
					err = os.Setenv("NO_OS_EXIT", "1")
					Expect(err).NotTo(HaveOccurred())
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())

					integration.RecursiveCopy(testInputPath, testOutputPath)

					// the test needs to execute in the same directory throughout the lifecycle of `ship update`
					testInputPath = testOutputPath

					err = os.Chdir(testOutputPath)
					Expect(err).NotTo(HaveOccurred())

					// read the test metadata
					testMetadata = readMetadata(testPath)
				}, 20)

				AfterEach(func() {
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						err := state.GetSingleton().RemoveStateFile()
						Expect(err).NotTo(HaveOccurred())

						// remove the temporary directory
						err = os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}
					err = os.Chdir(integrationDir)
					Expect(err).NotTo(HaveOccurred())
				}, 20)

				It("Should output files matching those expected when running in update mode", func() {
					if testMetadata.Skip {
						return
					}

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					err = os.Setenv("PRELOAD_TEST_STATE", "1")
					Expect(err).NotTo(HaveOccurred())
					args := []string{
						"update",
						"--headless",
						fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")),
					}
					args = append(args, testMetadata.Args...)
					cmd.SetArgs(args)
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					ignoreEntitlementSig := map[string][]string{
						".ship/state.json": {
							"v1.upstreamContents.appRelease.configSpec",
							"v1.upstreamContents.appRelease.entitlementSpec",
							"v1.upstreamContents.appRelease.entitlements",
							"v1.upstreamContents.appRelease.registrySecret",
							"v1.upstreamContents.appRelease.analyzeSpec",
							"v1.upstreamContents.appRelease.collectSpec",
							"v1.shipVersion",
						},
						".ship/upstream/appRelease.json": {
							"configSpec",
							"entitlementSpec",
							"entitlements",
							"registrySecret",
							"analyzeSpec",
							"collectSpec",
						},
					}

					// compare the files in the temporary directory with those in the "expected" directory
					// TODO: text based comparison of state files is brittle because helm values are being merged.
					// they should really be compared using the versioned state object
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, map[string]string{}, testMetadata.IgnoredFiles, ignoreEntitlementSig)
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
