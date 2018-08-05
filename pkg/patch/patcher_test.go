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

const createTestCasesFolder = "create-test-cases"
const mergeTestCasesFolder = "merge-test-cases"

var shipPatcher *ShipPatcher

var _ = BeforeSuite(func() {
	shipPatcher = &ShipPatcher{
		Logger: &logger.TestLogger{T: GinkgoT()},
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
				Expect(patch).To(Equal(expectPatch))
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
})
