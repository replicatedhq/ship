package patch

import (
	"io/ioutil"
	"path"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/pkg/testing/logger"
)

func TestShipPatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ShipPatcher")
}

const (
	createApplyTestCasesFolder = "create-apply-test-cases"
	mergeTestCasesFolder       = "merge-test-cases"
)

var shipPatcher *ShipPatcher

var _ = BeforeSuite(func() {
	shipPatcher = &ShipPatcher{
		Logger: &logger.TestLogger{T: GinkgoT()},
	}
})

var _ = Describe("ShipPatcher", func() {
	Describe("CreateTwoWayMergePatch", func() {
		It("Creates a merge patch given valid original and modified k8s yaml", func() {
			createTestDirs, err := ioutil.ReadDir(path.Join(createApplyTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, createTestDir := range createTestDirs {
				original, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, createTestDir.Name(), "original.yaml"))
				Expect(err).NotTo(HaveOccurred())

				modified, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, createTestDir.Name(), "modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				patch, err := shipPatcher.CreateTwoWayMergePatch(string(original), string(modified))
				Expect(err).NotTo(HaveOccurred())

				expectPatch, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, createTestDir.Name(), "patch.yaml"))
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
			applyTestDirs, err := ioutil.ReadDir(path.Join(createApplyTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, applyTestDir := range applyTestDirs {
				original, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, applyTestDir.Name(), "original.yaml"))
				Expect(err).NotTo(HaveOccurred())

				patch, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, applyTestDir.Name(), "patch.yaml"))
				Expect(err).NotTo(HaveOccurred())

				modified, err := shipPatcher.ApplyPatch(string(original), string(patch))
				Expect(err).NotTo(HaveOccurred())

				expectModified, err := ioutil.ReadFile(path.Join(createApplyTestCasesFolder, applyTestDir.Name(), "modified.yaml"))
				Expect(string(modified)).To(Equal(string(expectModified)))
			}
		})
	})
})
