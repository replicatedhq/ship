package integration

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
	"github.com/replicatedhq/ship/pkg/cli"
)

func TestCore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "integration")
}

// CompareDir returns false if the two directories have different contents
func CompareDir(expected, actual string) (bool, error) {
	expectedDir, err := ioutil.ReadDir(expected)
	Expect(err).NotTo(HaveOccurred())

	actualDir, err := ioutil.ReadDir(actual)
	Expect(err).NotTo(HaveOccurred())

	Expect(len(actualDir)).To(Equal(len(expectedDir)), fmt.Sprintf("Number of files in %s and %s differed", expected, actual))

	expectedMap := make(map[string]os.FileInfo)
	for _, file := range expectedDir {
		expectedMap[file.Name()] = file
	}

	actualMap := make(map[string]os.FileInfo)
	for _, file := range expectedDir {
		actualMap[file.Name()] = file
	}

	for name, expectedFile := range expectedMap {
		actualFile, ok := actualMap[name]
		Expect(ok).To(BeTrue())
		Expect(actualFile.IsDir()).To(Equal(expectedFile.IsDir()))

		expectedFilePath := filepath.Join(expected, expectedFile.Name())
		actualFilePath := filepath.Join(actual, actualFile.Name())

		if expectedFile.IsDir() {
			// compare child items
			result, err := CompareDir(expectedFilePath, actualFilePath)
			if !result || err != nil {
				return result, err
			}
		} else {
			// compare expectedFile contents
			expectedContents, err := ioutil.ReadFile(expectedFilePath)
			Expect(err).NotTo(HaveOccurred())
			actualContents, err := ioutil.ReadFile(actualFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(actualContents)).To(Equal(string(expectedContents)), fmt.Sprintf("Contents of files %s (expected) and %s (actual) did not match", expectedFilePath, actualFilePath))
		}
	}

	return true, nil
}

var _ = Describe("basic", func() {
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
				testOutputPath := path.Join(testPath, "tmp")
				testInputPath := path.Join(testPath, "input")

				var customerID, installationID, releaseVersion string

				BeforeEach(func() {
					// create a temporary directory within this directory to compare files with
					os.RemoveAll(testOutputPath)
					err := os.Mkdir(testOutputPath, os.ModeDir|os.ModePerm)
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(testOutputPath)

					// read the customer ID, installation ID and release version for this test
					custIDBytes, err := ioutil.ReadFile(path.Join(testPath, "customer_id"))
					Expect(err).NotTo(HaveOccurred())
					customerID = string(custIDBytes)

					installationIDBytes, err := ioutil.ReadFile(path.Join(testPath, "installation_id"))
					Expect(err).NotTo(HaveOccurred())
					installationID = string(installationIDBytes)

					releaseVersionBytes, err := ioutil.ReadFile(path.Join(testPath, "release_version"))
					Expect(err).NotTo(HaveOccurred())
					releaseVersion = string(releaseVersionBytes)
				})

				AfterEach(func() {
					// remove the temporary directory
					err := os.RemoveAll(testOutputPath)
					Expect(err).NotTo(HaveOccurred())
					os.Chdir(integrationDir)
				})

				It("Should output files matching those expected when running in local mode", func() {
					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs([]string{
						"--headless",
						fmt.Sprintf("--studio-file=%s", path.Join(testInputPath, ".ship/release.yml")),
						fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")),
						"--log-level=off",
					})
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					//compare the files in the temporary directory with those in the "expected" directory
					result, err := CompareDir(path.Join(testPath, "expected"), testOutputPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())

				}, 60)

				It("Should output files matching those expected when communicating with the graphql api", func() {
					cmd := cli.RootCmd()
					buf := new(bytes.Buffer)
					cmd.SetOutput(buf)
					cmd.SetArgs(append([]string{
						"--headless",
						fmt.Sprintf("--state-file=%s", path.Join(testInputPath, ".ship/state.json")),
						"--customer-endpoint=https://pg.staging.replicated.com/graphql",
						"--log-level=off",
						fmt.Sprintf("--customer-id=%s", customerID),
						fmt.Sprintf("--installation-id=%s", installationID),
						fmt.Sprintf("--release-semver=%s", releaseVersion),
					}))
					err := cmd.Execute()
					Expect(err).NotTo(HaveOccurred())

					//compare the files in the temporary directory with those in the "expected" directory
					result, err := CompareDir(path.Join(testPath, "expected"), testOutputPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeTrue())

				}, 60)
			})
		}
	}
})
