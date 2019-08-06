package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"
)

// files and directories with non-deterministic output
var skipFiles = []string{
	"installer/terraform/.terraform/plugins",
	"installer/terraform/plan.tfplan",
	"installer/charts/rendered/secrets.yaml",
	"installer/base/consul-test.yaml",
	"installer/base/gossip-secret.yaml",
	"installer/consul-rendered.yaml",
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
func CompareDir(expected, actual string, replacements map[string]string, ignoredFiles []string, ignoredKeys map[string][]string) (bool, error) {
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
				cwd, err := os.Getwd()
				Expect(err).NotTo(HaveOccurred())

				relativeActualFilePath, err := filepath.Rel(cwd, actualFilePath)
				Expect(err).NotTo(HaveOccurred())

				fileIgnorePaths := ignoredKeys[relativeActualFilePath]

				expectedContentsBytes, err = prettyAndCleanJSON(expectedContentsBytes, fileIgnorePaths)
				Expect(err).NotTo(HaveOccurred())

				actualContentsBytes, err = prettyAndCleanJSON(actualContentsBytes, fileIgnorePaths)
				Expect(err).NotTo(HaveOccurred())
			}

			// kind of a hack -- remove any trailing newlines (because text editors are hard to use)
			expectedContents := strings.TrimRight(string(expectedContentsBytes), "\n")
			actualContents := strings.TrimRight(string(actualContentsBytes), "\n")

			// find and replace strings from the expected contents (customerID, installationID, etc)
			for k, v := range replacements {
				re := regexp.MustCompile(k)
				expectedContents = re.ReplaceAllString(expectedContents, v)
			}

			// find and replace strings from the actual contents (datetime, signature, etc)
			for k, v := range replacements {
				re := regexp.MustCompile(k)
				actualContents = re.ReplaceAllString(actualContents, v)
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

func prettyAndCleanJSON(data []byte, keysToIgnore []string) ([]byte, error) {
	var obj interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}

	if _, ok := obj.(map[string]interface{}); ok && keysToIgnore != nil {
		for _, key := range keysToIgnore {
			obj = replaceInJSON(obj.(map[string]interface{}), key)
		}
	}

	data, err = json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "marshal")
	}
	return data, nil
}

func replaceInJSON(obj map[string]interface{}, path string) map[string]interface{} {
	// split path on '.'

	if path == "" {
		return obj
	}

	fullpath := strings.Split(path, ".")

	// if the object to delete is at this level, delete it from the map and return
	if len(fullpath) == 1 {
		delete(obj, fullpath[0])
		return obj
	}

	// the object to delete is at a deeper level - check if the specified key exists, if it does not then return
	// else recursively call replaceInJSON

	if _, exists := obj[fullpath[0]]; !exists {
		// the object to delete does not exist
		return obj
	}

	subObj, ok := obj[fullpath[0]].(map[string]interface{})
	if !ok {
		fmt.Printf("looking for %q and this is not a map[string]interface{}: %+v\n", path, obj[fullpath[0]])
		delete(obj, fullpath[0])
		return obj
	}

	replacedObj := replaceInJSON(subObj, strings.Join(fullpath[1:], "."))
	obj[fullpath[0]] = replacedObj

	if len(replacedObj) == 0 {
		delete(obj, fullpath[0])
	}

	return obj
}

func RecursiveCopy(sourceDir, destDir string) {
	err := os.MkdirAll(destDir, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())
	srcFiles, err := ioutil.ReadDir(sourceDir)
	Expect(err).NotTo(HaveOccurred())
	for _, file := range srcFiles {
		if file.IsDir() {
			RecursiveCopy(filepath.Join(sourceDir, file.Name()), filepath.Join(destDir, file.Name()))
		} else {
			// is file
			contents, err := ioutil.ReadFile(filepath.Join(sourceDir, file.Name()))
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(filepath.Join(destDir, file.Name()), contents, file.Mode())
			Expect(err).NotTo(HaveOccurred())
		}
	}
}
