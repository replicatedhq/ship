package patch

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/afero"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/process"
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
)

var shipPatcher *ShipPatcher

var _ = BeforeSuite(func() {
	logger := &logger.TestLogger{T: GinkgoT()}
	shipPatcher = &ShipPatcher{
		Logger:  logger,
		FS:      afero.Afero{Fs: afero.NewOsFs()},
		process: process.Process{Logger: logger},
		cmd:     exec.Command(os.Args[0], "-test.run=TestMockKustomize"),
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

				_, err = shipPatcher.ApplyPatch(string(patch))
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})

func TestMockKustomize(t *testing.T) {
	// this test does nothing when run normally, only when
	// invoked by other tests. Those tests should set this
	// env var in order to get the behavior
	if os.Getenv("GOTEST_SUBPROCESS_MOCK") == "" {
		return
	}

	receivedArgs := os.Args[2:]
	expectTemplate := []string{"template", constants.TempApplyOverlayPath}
	if reflect.DeepEqual(receivedArgs, expectTemplate) {
		// we good, these are exepcted calls, and we just need to test one type of forking
		os.Exit(0)
	}

	if os.Getenv("CRASHING_KUSTOMIZE_ERROR") != "" {
		fmt.Fprintf(os.Stdout, os.Getenv("CRASHING_KUSTOMIZE_ERROR"))
		os.Exit(1)
	}

	if os.Getenv("EXPECT_KUSTOMIZE_ARGV") != "" {
		// this is janky, but works for our purposes, use pipe | for separator, since its unlikely to be in argv
		expectedArgs := strings.Split(os.Getenv("EXPECT_KUSTOMIZE_ARGV"), "|")

		fmt.Fprintf(os.Stderr, "expected args %v, got args %v", expectedArgs, receivedArgs)
		if !reflect.DeepEqual(receivedArgs, expectedArgs) {
			fmt.Fprint(os.Stderr, "; FAIL")
			os.Exit(2)
		}

		os.Exit(0)
	}
}
