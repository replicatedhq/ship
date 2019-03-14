package patch

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	k8stypes "sigs.k8s.io/kustomize/pkg/types"
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

				patch, err := shipPatcher.CreateTwoWayMergePatch(original, modified)
				Expect(err).NotTo(HaveOccurred())

				expectPatch, err := ioutil.ReadFile(path.Join(createTestCasesFolder, createTestDir.Name(), "patch.yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(patch)).To(Equal(string(expectPatch)))
			}
		})
	})
	Describe("MergePatches", func() {
		mergePatchPathMap := map[string][]string{
			"basic": {"spec", "template", "spec", "containers", "0", "name"},
			"list":  {"spec", "template", "spec", "containers", "0", "env", "2", "value"},
		}
		It("Creates a single patch with the effect of both given patches", func() {
			mergeTestDirs, err := ioutil.ReadDir(path.Join(mergeTestCasesFolder))
			Expect(err).NotTo(HaveOccurred())

			for _, mergeTestDir := range mergeTestDirs {
				original, err := ioutil.ReadFile(path.Join(mergeTestCasesFolder, mergeTestDir.Name(), "patch.yaml"))
				Expect(err).NotTo(HaveOccurred())

				expectPatch, err := ioutil.ReadFile(path.Join(mergeTestCasesFolder, mergeTestDir.Name(), "modified.yaml"))
				Expect(err).NotTo(HaveOccurred())

				err = os.Chdir(path.Join(mergeTestCasesFolder, mergeTestDir.Name()))
				Expect(err).NotTo(HaveOccurred())

				patch, err := shipPatcher.MergePatches(
					original,
					mergePatchPathMap[mergeTestDir.Name()],
					api.Kustomize{Base: "base"},
					"base/deployment.yaml",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(patch)).To(Equal(string(expectPatch)))
				Expect(os.Chdir("../..")).NotTo(HaveOccurred())
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

				modified, err := shipPatcher.ApplyPatch(patch, api.Kustomize{Base: "base"}, "base/deployment.yaml")
				Expect(err).NotTo(HaveOccurred())

				Expect(modified).To(Equal(expectModified))
				Expect(os.Chdir("../..")).NotTo(HaveOccurred())
			}
		})
	})
	Describe("ModifyField", func() {
		modifyFieldPathMap := map[string][]string{
			"basic":  {"spec", "template", "spec", "containers", "0", "name"},
			"list":   {"spec", "template", "spec", "containers", "0", "ports", "1", "name"},
			"nested": {"spec", "template", "spec", "containers", "0", "env", "0", "valueFrom", "configMapKeyRef", "key"},
			"nil":    {"spec", "template", "spec", "containers", "0", "volumeMounts", "1", "mountPath"},
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

				modified, err := shipPatcher.ModifyField(originalFile, pathToModify)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(modified)).To(Equal(string(expectModified)))
			}
		})
	})
})

func TestShipPatcher_writeTempKustomization(t *testing.T) {
	type testFile struct {
		path     string
		contents string
	}
	tests := []struct {
		name                string
		step                api.Kustomize
		resource            string
		testFiles           []testFile
		expectKustomization k8stypes.Kustomization
		expectErr           bool
	}{
		{
			name:     "no matching resource",
			step:     api.Kustomize{Base: "base/"},
			resource: "./base/file.yaml",
			testFiles: []testFile{
				{
					path: "./base/strawberry.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry`,
				},
			},
			expectErr: true,
		},
		{
			name:     "matching resource",
			step:     api.Kustomize{Base: "base/"},
			resource: "./base/strawberry.yaml",
			testFiles: []testFile{
				{
					path: "base/strawberry.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry`,
				},
			},
			expectErr: false,
			expectKustomization: k8stypes.Kustomization{
				Resources: []string{"strawberry.yaml"},
			},
		},
		{
			name:     "matching resource, unclean path",
			step:     api.Kustomize{Base: "base/"},
			resource: "./base/strawberry.yaml",
			testFiles: []testFile{
				{
					path: "./base/strawberry.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry`,
				},
			},
			expectErr: false,
			expectKustomization: k8stypes.Kustomization{
				Resources: []string{"strawberry.yaml"},
			},
		},
		{
			name:     "matching resource, unclean path in subdir",
			step:     api.Kustomize{Base: "base/"},
			resource: "./base/flowers/rose.yml",
			testFiles: []testFile{
				{
					path: "./base/strawberry.yaml",
					contents: `apiVersion: apps/v1beta2
kind: Deployment
metadata:
  labels:
    app: strawberry
    heritage: Tiller
    chart: strawberry-1.0.0
  name: strawberry`,
				},
				{
					path: "./base/flowers/rose.yml",
					contents: `apiVersion: v1
kind: Service
metadata:
  labels:
    app: rose
  name: rose`,
				},
			},
			expectErr: false,
			expectKustomization: k8stypes.Kustomization{
				Resources: []string{"flowers/rose.yml"},
			},
		},
		{
			name:     "alternate base",
			step:     api.Kustomize{Base: "another/base/path/"},
			resource: "another/base/path/raspberry.yaml",
			testFiles: []testFile{
				{
					path: "another/base/path/raspberry.yaml",
					contents: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: raspberry
  name: raspberry`,
				},
			},
			expectErr: false,
			expectKustomization: k8stypes.Kustomization{
				Resources: []string{"raspberry.yaml"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			for _, testFile := range tt.testFiles {
				err := mockFs.WriteFile(testFile.path, []byte(testFile.contents), 0755)
				req.NoError(err)
			}
			p := &ShipPatcher{
				Logger: log.NewNopLogger(),
				FS:     mockFs,
			}

			err := p.writeTempKustomization(tt.step, tt.resource)

			if !tt.expectErr {
				req.NoError(err)

				kustomizationB, err := mockFs.ReadFile(path.Join(tt.step.Base, "kustomization.yaml"))
				req.NoError(err)

				kustomizationYaml := k8stypes.Kustomization{}
				err = yaml.Unmarshal(kustomizationB, &kustomizationYaml)
				req.NoError(err)
				req.Equal(tt.expectKustomization, kustomizationYaml)
			} else {
				req.Error(err)
			}
		})
	}
}
