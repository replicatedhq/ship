package patch

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/spf13/afero"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
)

func TestShipPatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ShipPatcher")
}

const (
	createTestCasesFolder = "create-test-cases"
	mergeTestCasesFolder  = "merge-test-cases"
	applyTestCasesFolder  = "apply-test-cases"
	modifyTestCasesFolder = "modify-test-cases"
)

var shipPatcher *ShipPatcher

var _ = BeforeSuite(func() {
	logger := &logger.TestLogger{T: GinkgoT()}
	shipPatcher = &ShipPatcher{
		Logger: logger,
		FS:     afero.Afero{Fs: afero.NewOsFs()},
	}
})

var _ = Describe("ShipPatcher", func() {
	Describe("CreateTwoWayMergePatch", func() {
		It("Creates a merge patch given valid original and modified k8s yaml", func() {
			createTestDirs, err := ioutil.ReadDir(path.Join(createTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, createTestDir := range createTestDirs {
				original, err := ioutil.ReadFile(path.Join(createTestCasesFolder, createTestDir.Name(), "original.yaml"))
				Expect(err).NotTo(HaveOccurred())

				modified, err := ioutil.ReadFile(path.Join(createTestCasesFolder, createTestDir.Name(), "modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				patch, err := shipPatcher.CreateTwoWayMergePatch(string(original), string(modified))
				Expect(err).NotTo(HaveOccurred())

				expectPatch, err := ioutil.ReadFile(path.Join(createTestCasesFolder, createTestDir.Name(), "patch.yaml"))
				Expect(string(patch)).To(Equal(string(expectPatch)))
			}
		})
	})
	Describe("MergePatches", func() {
		It("Creates a single patch with the effect of both given patches", func() {
			mergeTestDirs, err := ioutil.ReadDir(path.Join(mergeTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, mergeTestDir := range mergeTestDirs {
				original, err := ioutil.ReadFile(path.Join(mergeTestCasesFolder, mergeTestDir.Name(), "original.yaml"))
				Expect(err).NotTo(HaveOccurred())

				modified, err := ioutil.ReadFile(path.Join(mergeTestCasesFolder, mergeTestDir.Name(), "modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				patch, err := shipPatcher.MergePatches(original, modified)
				Expect(err).NotTo(HaveOccurred())

				expectPatch, err := ioutil.ReadFile(path.Join(mergeTestCasesFolder, mergeTestDir.Name(), "patch.yaml"))
				Expect(patch).To(Equal(expectPatch))
			}
		})
	})
	Describe("ApplyPatch", func() {
		It("Applies a single patch to a file, producing a modified yaml", func() {
			applyTestDirs, err := ioutil.ReadDir(path.Join(applyTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, applyTestDir := range applyTestDirs {
				err := os.Chdir(path.Join(applyTestCasesFolder, applyTestDir.Name()))
				Expect(err).NotTo(HaveOccurred())

				patch, err := ioutil.ReadFile(path.Join("patch.yaml"))
				Expect(err).NotTo(HaveOccurred())

				expectModified, err := ioutil.ReadFile(path.Join("modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				modified, err := shipPatcher.ApplyPatch(string(patch), api.Kustomize{BasePath: "base"}, "base/deployment.yaml")
				Expect(err).NotTo(HaveOccurred())

				Expect(modified).To(Equal(expectModified))
			}
			os.Chdir("../../")
		})
	})
	Describe("ModifyField", func() {
		modifyFieldPathMap := map[string][]string{
			"basic":  []string{"spec", "template", "spec", "containers", "0", "name"},
			"list":   []string{"spec", "template", "spec", "containers", "0", "ports", "1", "name"},
			"nested": []string{"spec", "template", "spec", "containers", "0", "env", "0", "valueFrom", "configMapKeyRef", "key"},
		}
		It("Modifies a single field in yaml with PATCH_TOKEN", func() {
			modifyTestDirs, err := ioutil.ReadDir(path.Join(modifyTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, modifyTestDir := range modifyTestDirs {
				originalFile, err := ioutil.ReadFile(path.Join(modifyTestCasesFolder, modifyTestDir.Name(), "original.yaml"))
				Expect(err).NotTo(HaveOccurred())

				expectModified, err := ioutil.ReadFile(path.Join(modifyTestCasesFolder, modifyTestDir.Name(), "modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				pathToModify, ok := modifyFieldPathMap[modifyTestDir.Name()]
				Expect(ok).To(BeTrue())

				modified, err := shipPatcher.ModifyField(string(originalFile), pathToModify)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(modified)).To(Equal(string(expectModified)))
			}
		})
	})
})
