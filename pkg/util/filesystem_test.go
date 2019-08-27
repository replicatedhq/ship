package util

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/api"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util")
}

type testFile struct {
	path     string
	contents string
}

func addTestFiles(fs afero.Afero, testFiles []testFile) error {
	for _, testFile := range testFiles {
		if err := fs.MkdirAll(filepath.Dir(testFile.path), 0755); err != nil {
			return err
		}
		if err := fs.WriteFile(testFile.path, []byte(testFile.contents), 0644); err != nil {
			return err
		}
	}
	return nil
}

func readTestFiles(step api.Kustomize, fs afero.Afero) ([]testFile, error) {
	files := []testFile{}
	if err := fs.Walk(step.Base, func(targetPath string, info os.FileInfo, err error) error {
		if filepath.Ext(targetPath) == ".yaml" {
			contents, err := fs.ReadFile(targetPath)
			if err != nil {
				return err
			}

			files = append(files, testFile{
				path:     targetPath,
				contents: string(contents),
			})
		}
		return nil
	}); err != nil {
		return files, err
	}

	return files, nil
}

var _ = Describe("Util", func() {
	Describe("FindOnlySubdir", func() {
		Context("Provided a path with a single subdirectory", func() {
			It("returns the file path to the subdirectory", func() {
				singleSubDirPath := "singleSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				err := mockFs.Mkdir(filepath.Join(singleSubDirPath, "test"), 0755)
				Expect(err).NotTo(HaveOccurred())

				dirPath, err := FindOnlySubdir(singleSubDirPath, mockFs)
				Expect(err).NotTo(HaveOccurred())
				Expect(dirPath).To(Equal(filepath.Join(singleSubDirPath, "test")))
			})
		})
		Context("Provided a path with multiple subdirectories", func() {
			It("returns an error", func() {
				multipleSubDirPath := "multipleSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				err := mockFs.Mkdir(filepath.Join(multipleSubDirPath, "test"), 0755)
				Expect(err).NotTo(HaveOccurred())
				err = mockFs.Mkdir(filepath.Join(multipleSubDirPath, "test2"), 0755)
				Expect(err).NotTo(HaveOccurred())

				_, err = FindOnlySubdir(multipleSubDirPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with a no subdirectories", func() {
			It("returns an error", func() {
				noSubDirPath := "noSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				err := mockFs.Mkdir(filepath.Join(noSubDirPath), 0755)
				Expect(err).NotTo(HaveOccurred())

				_, err = FindOnlySubdir(noSubDirPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with only files and no subdirectories", func() {
			It("returns the file path to the subdirectory", func() {
				onlyFilesPath := "onlyFilesPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				err := mockFs.Mkdir(filepath.Join(onlyFilesPath), 0755)
				Expect(err).NotTo(HaveOccurred())
				err = mockFs.WriteFile(filepath.Join(onlyFilesPath, "test"), []byte("hi"), 0644)
				Expect(err).NotTo(HaveOccurred())
				err = mockFs.WriteFile(filepath.Join(onlyFilesPath, "test2"), []byte("bye"), 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = FindOnlySubdir(onlyFilesPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with files and a single subdirectory", func() {
			It("returns the file path to the subdirectory", func() {
				filesAndSubDirPath := "filesAndSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				err := mockFs.MkdirAll(filepath.Join(filesAndSubDirPath, "testSubDir"), 0755)
				Expect(err).NotTo(HaveOccurred())
				err = mockFs.WriteFile(filepath.Join(filesAndSubDirPath, "test"), []byte("hi"), 0644)
				Expect(err).NotTo(HaveOccurred())
				err = mockFs.WriteFile(filepath.Join(filesAndSubDirPath, "test2"), []byte("bye"), 0644)
				Expect(err).NotTo(HaveOccurred())

				dirPath, _ := FindOnlySubdir(filesAndSubDirPath, mockFs)
				Expect(dirPath).To(Equal(filepath.Join(filesAndSubDirPath, "testSubDir")))
			})
		})
	})
})

func TestRecursiveCopy(t *testing.T) {
	tests := []struct {
		name        string
		sourceFiles []testFile
		expectFiles []testFile
		sourceDir   string
		destDir     string
		wantErr     bool
	}{
		{
			name: "basic",
			sourceFiles: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			expectFiles: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			sourceDir: "abc",
			destDir:   "xyz",
			wantErr:   false,
		},
		{
			name: "recursive",
			sourceFiles: []testFile{
				{
					path: "abc/abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			expectFiles: []testFile{
				{
					path: "abc/abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "xyz/abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			sourceDir: "abc",
			destDir:   "xyz",
			wantErr:   false,
		},
		{
			name: "multiple files and dirs",
			sourceFiles: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "abc/apple.yaml",
					contents: `kind: Fruit
metadata:
  name: apple
spec:
  original: a generic apple
`,
				},
				{
					path: "abc/xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			expectFiles: []testFile{
				{
					path: "abc/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "abc/apple.yaml",
					contents: `kind: Fruit
metadata:
  name: apple
spec:
  original: a generic apple
`,
				},
				{
					path: "abc/xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},

				{
					path: "xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
				{
					path: "xyz/apple.yaml",
					contents: `kind: Fruit
metadata:
  name: apple
spec:
  original: a generic apple
`,
				},
				{
					path: "xyz/xyz/dragonfruit.yaml",
					contents: `kind: Fruit
metadata:
  name: dragonfruit
spec:
  original: original dragonfruit
`,
				},
			},
			sourceDir: "abc",
			destDir:   "xyz",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			err := addTestFiles(mockFs, tt.sourceFiles)
			req.NoError(err)

			err = RecursiveCopy(mockFs, tt.sourceDir, tt.destDir)
			if tt.wantErr {
				req.Error(err)
				return
			} else {
				req.NoError(err)
			}

			step := api.Kustomize{Base: ""}
			actual, err := readTestFiles(step, mockFs)
			req.NoError(err)

			req.ElementsMatch(tt.expectFiles, actual)
		})
	}
}
