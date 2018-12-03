package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/pmezard/go-difflib/difflib"
)

// files and directories with non-deterministic output
var skipFiles = []string{
	"installer/terraform/.terraform/plugins",
	"installer/terraform/plan.tfplan",
	"installer/charts/rendered/secrets.yaml",
}

func skipCheck(filePath string, ignoredFiles []string) bool {
	for _, f := range ignoredFiles {
		if strings.HasSuffix(filePath, f) {
			return true
		}
	}

	for _, f := range skipFiles {
		if strings.HasSuffix(filePath, f) {
			return true
		}
	}
	return false
}

// CompareDir returns false if the two directories have different contents
func CompareDir(expected, actual string, replacements map[string]string, ignoredFiles []string, ignoredKeys []map[string][]string) (bool, error) {
	if skipCheck(actual, ignoredFiles) {
		return true, nil
	}

	expectedDir, err := ioutil.ReadDir(expected)
	Expect(err).NotTo(HaveOccurred())

	actualDir, err := ioutil.ReadDir(actual)
	Expect(err).NotTo(HaveOccurred())

	expectedMap := make(map[string]os.FileInfo)
	expectedFilenamesMap := make(map[string]struct{})
	for _, file := range expectedDir {
		expectedMap[file.Name()] = file
		expectedFilenamesMap[file.Name()] = struct{}{}
	}

	actualMap := make(map[string]os.FileInfo)
	actualFilenamesMap := make(map[string]struct{})
	for _, file := range actualDir {
		actualMap[file.Name()] = file
		actualFilenamesMap[file.Name()] = struct{}{}
	}

	Expect(actualFilenamesMap).To(Equal(expectedFilenamesMap), fmt.Sprintf("Contents of directories %s (expected) and %s (actual) did not match", expected, actual))

	for name, expectedFile := range expectedMap {
		actualFile, ok := actualMap[name]
		Expect(ok).To(BeTrue())
		Expect(actualFile.IsDir()).To(Equal(expectedFile.IsDir()))

		expectedFilePath := filepath.Join(expected, expectedFile.Name())
		actualFilePath := filepath.Join(actual, actualFile.Name())

		if expectedFile.IsDir() {
			// compare child items
			result, err := CompareDir(expectedFilePath, actualFilePath, replacements, ignoredFiles, ignoredKeys)
			if !result || err != nil {
				return result, err
			}
		} else if skipCheck(expectedFilePath, ignoredFiles) {
			continue
		} else {
			// compare expectedFile contents
			expectedContentsBytes, err := ioutil.ReadFile(expectedFilePath)
			Expect(err).NotTo(HaveOccurred())
			actualContentsBytes, err := ioutil.ReadFile(actualFilePath)
			Expect(err).NotTo(HaveOccurred())

			// another hack for ease of testing -- pretty print json before comparing so diffs
			// are easier to read
			if strings.HasSuffix(actualFilePath, ".json") {
				expectedContentsBytes = prettyAndCleanJSON(expectedContentsBytes, nil)
				actualContentsBytes = prettyAndCleanJSON(actualContentsBytes, nil)

				cwd, err := os.Getwd()
				Expect(err).NotTo(HaveOccurred())

				for _, paths := range ignoredKeys {
					for path := range paths {
						relativeActualFilePath, err := filepath.Rel(cwd, actualFilePath)
						Expect(err).NotTo(HaveOccurred())

						if path == relativeActualFilePath {
							expectedContentsBytes = prettyAndCleanJSON(expectedContentsBytes, paths[path])
							actualContentsBytes = prettyAndCleanJSON(actualContentsBytes, paths[path])
						}
					}
				}

			}

			// kind of a hack -- remove any trailing newlines (because text editors are hard to use)
			expectedContents := strings.TrimRight(string(expectedContentsBytes), "\n")
			actualContents := strings.TrimRight(string(actualContentsBytes), "\n")

			// find and replace strings from the expected contents (customerID, installationID, etc)
			for k, v := range replacements {
				expectedContents = strings.Replace(expectedContents, k, v, -1)
			}

			diff := difflib.UnifiedDiff{
				A:        difflib.SplitLines(expectedContents),
				B:        difflib.SplitLines(actualContents),
				FromFile: "expected contents",
				ToFile:   "actual contents",
				Context:  3,
			}

			diffText, err := difflib.GetUnifiedDiffString(diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(diffText).To(BeEmpty(), fmt.Sprintf("Contents of files %s (expected) and %s (actual) did not match", expectedFilePath, actualFilePath))
		}
	}

	return true, nil
}

func prettyAndCleanJSON(data []byte, keysToIgnore []string) []byte {
	var obj interface{}
	err := json.Unmarshal(data, &obj)
	Expect(err).NotTo(HaveOccurred())

	data, err = json.MarshalIndent(obj, "", "  ")
	Expect(err).NotTo(HaveOccurred())

	return data
}
