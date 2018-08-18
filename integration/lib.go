package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"encoding/json"

	. "github.com/onsi/gomega"
	"github.com/pmezard/go-difflib/difflib"
)

// files and directories with non-deterministic output
var skipFiles = []string{
	"installer/terraform/.terraform/plugins",
	"installer/terraform/plan",
	"installer/terraform/terraform.tfstate",
	"installer/charts/rendered/secrets.yaml",
	"base/secrets.yaml",
}

func skipCheck(filepath string) bool {
	for _, f := range skipFiles {
		if strings.HasSuffix(filepath, f) {
			return true
		}
	}
	return false
}

// CompareDir returns false if the two directories have different contents
func CompareDir(expected, actual string) (bool, error) {
	if skipCheck(actual) {
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
			result, err := CompareDir(expectedFilePath, actualFilePath)
			if !result || err != nil {
				return result, err
			}
		} else if skipCheck(expectedFilePath) {
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
				var obj interface{}
				err = json.Unmarshal(expectedContentsBytes, &obj)
				Expect(err).NotTo(HaveOccurred())
				expectedContentsBytes, err = json.MarshalIndent(obj, "", "  ")

				obj = nil
				err = json.Unmarshal(actualContentsBytes, &obj)
				Expect(err).NotTo(HaveOccurred())
				actualContentsBytes, err = json.MarshalIndent(obj, "", "  ")
			}

			// kind of a hack -- remove any trailing newlines (because text editors are hard to use)
			expectedContents := strings.TrimRight(string(expectedContentsBytes), "\n")
			actualContents := strings.TrimRight(string(actualContentsBytes), "\n")

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
