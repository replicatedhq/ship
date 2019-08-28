package base

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	LicenseID      string            `yaml:"license_id"`
	AppSlug        string            `yaml:"app_slug"`
	InstallationID string            `yaml:"installation_id"`
	CustomerID     string            `yaml:"customer_id"`
	ReleaseVersion string            `yaml:"release_version"`
	SetChannelName string            `yaml:"set_channel_name"`
	Flavor         string            `yaml:"flavor"`
	Args           []string          `yaml:"args"`
	Replacements   map[string]string `yaml:"replacements"`

	// debugging
	SkipCleanup bool `yaml:"skip_cleanup"`
	SkipEdit    bool `yaml:"skip_edit"`
	SkipInit    bool `yaml:"skip_init"`
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
			When(fmt.Sprintf("the spec in %q is run with shipinit", file.Name()), func() {
				testPath := path.Join(integrationDir, file.Name())
				var testOutputPath string
				var testMetadata TestMetadata

				BeforeEach(func(done chan<- interface{}) {
					err = os.Setenv("NO_OS_EXIT", "1")
					Expect(err).NotTo(HaveOccurred())
					err = os.Setenv("REPLICATED_REGISTRY", "registry.staging.replicated.com")
					Expect(err).NotTo(HaveOccurred())
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					err = os.Chdir(testOutputPath)
					Expect(err).NotTo(HaveOccurred())

					// read the test metadata
					testMetadata = readMetadata(testPath)

					if testMetadata.Replacements == nil {
						testMetadata.Replacements = make(map[string]string)
					}

					// TODO - instead of getting installation ID, etc from test metadata create a release with the vendor api
					// TODO customer ID and vendor token will need to be read from environment variables
					// TODO so will the desired environment - staging vs prod

					close(done)
				}, 20)

				AfterEach(func() {
					err = os.Unsetenv("REPLICATED_REGISTRY")
					Expect(err).NotTo(HaveOccurred())
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						err := state.GetSingleton().RemoveStateFile()
						Expect(err).NotTo(HaveOccurred())

						// remove the temporary directory
						err = os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}

					err := state.GetSingleton().RemoveStateFile()
					Expect(err).NotTo(HaveOccurred())

					err = os.Chdir(integrationDir)
					Expect(err).NotTo(HaveOccurred())
				}, 20)

				It("Should output files matching those expected when communicating with the graphql api", func() {
					if testMetadata.SkipInit {
						Skip("this test case is set to skip init tests")
						return
					}

					upstream := "staging.replicated.app/some-cool-ci-tool"

					if testMetadata.InstallationID != "" {
						// this should probably be url encoded but whatever
						upstream = fmt.Sprintf(
							"%s?installation_id=%s&customer_id=%s",
							upstream,
							testMetadata.InstallationID,
							testMetadata.CustomerID,
						)
					} else {
						upstream = fmt.Sprintf(
							"staging.replicated.app/%s/?license_id=%s",
							testMetadata.AppSlug,
							testMetadata.LicenseID)
					}

					if testMetadata.ReleaseVersion != "" {
						upstream = fmt.Sprintf("%s&release_semver=%s", upstream, testMetadata.ReleaseVersion)
					}

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
					testMetadata.Replacements["__upstream__"] = strings.Replace(upstream, "&", "\\u0026", -1) // this string is encoded within the output
					testMetadata.Replacements["__installationID__"] = testMetadata.InstallationID
					testMetadata.Replacements["__customerID__"] = testMetadata.CustomerID
					testMetadata.Replacements["__appSlug__"] = testMetadata.AppSlug
					testMetadata.Replacements["__licenseID__"] = testMetadata.LicenseID

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
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, testMetadata.Replacements, []string{}, ignoreEntitlementSig)
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

	for _, file := range files {
		if file.IsDir() {
			When(fmt.Sprintf("the spec in %q is run with shipedit", file.Name()), func() {
				testPath := path.Join(integrationDir, file.Name())
				var testOutputPath string
				var testMetadata TestMetadata

				BeforeEach(func(done chan<- interface{}) {
					err = os.Setenv("NO_OS_EXIT", "1")
					Expect(err).NotTo(HaveOccurred())
					err = os.Setenv("REPLICATED_REGISTRY", "registry.staging.replicated.com")
					Expect(err).NotTo(HaveOccurred())
					// create a temporary directory within this directory to compare files with
					testOutputPath, err = ioutil.TempDir(testPath, "_test_")
					Expect(err).NotTo(HaveOccurred())
					err = os.Chdir(testOutputPath)
					Expect(err).NotTo(HaveOccurred())

					// read the test metadata
					testMetadata = readMetadata(testPath)

					if testMetadata.Replacements == nil {
						testMetadata.Replacements = make(map[string]string)
					}

					close(done)
				}, 20)

				AfterEach(func() {
					err = os.Unsetenv("REPLICATED_REGISTRY")
					Expect(err).NotTo(HaveOccurred())
					if !testMetadata.SkipCleanup && os.Getenv("SHIP_INTEGRATION_SKIP_CLEANUP_ALL") == "" {
						// remove the temporary directory
						err := os.RemoveAll(testOutputPath)
						Expect(err).NotTo(HaveOccurred())
					}

					err := state.GetSingleton().RemoveStateFile()
					Expect(err).NotTo(HaveOccurred())

					err = os.Chdir(integrationDir)
					Expect(err).NotTo(HaveOccurred())
				}, 20)

				It("Should output files matching those expected when running with ship edit", func() {
					if testMetadata.SkipEdit {
						Skip("this test case is set to skip edit tests")
						return
					}

					// copy the expected ship state to the output
					// and run any replacements needed

					integration.RecursiveCopy(filepath.Join(testPath, "expected", ".ship"), filepath.Join(testOutputPath, ".ship"))

					testMetadata.Replacements["__installationID__"] = testMetadata.InstallationID
					testMetadata.Replacements["__customerID__"] = testMetadata.CustomerID
					testMetadata.Replacements["__appSlug__"] = testMetadata.AppSlug
					testMetadata.Replacements["__licenseID__"] = testMetadata.LicenseID

					readPath := filepath.Join(testPath, "expected", ".ship", "state.json")
					stateFile, err := ioutil.ReadFile(readPath)
					Expect(err).NotTo(HaveOccurred())

					for k, v := range testMetadata.Replacements {
						stateFile = []byte(strings.Replace(string(stateFile), k, v, -1))
					}

					writePath := filepath.Join(testOutputPath, ".ship", "state.json")
					err = os.MkdirAll(filepath.Dir(writePath), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())
					err = ioutil.WriteFile(writePath, stateFile, os.ModePerm)
					Expect(err).NotTo(HaveOccurred())

					if state.GetSingleton() != nil {
						err = state.GetSingleton().ReloadFile()
						Expect(err).NotTo(HaveOccurred())
					}

					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"edit",
						"--headless",
						"--log-level=off",
					}, testMetadata.Args...))
					err = cmd.Execute()
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
					result, err := integration.CompareDir(path.Join(testPath, "expected"), testOutputPath, testMetadata.Replacements, []string{}, ignoreEntitlementSig)
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
