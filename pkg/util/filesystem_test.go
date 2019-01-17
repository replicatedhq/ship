package util

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util")
}

var _ = Describe("Util", func() {
	Describe("FindOnlySubdir", func() {
		Context("Provided a path with a single subdirectory", func() {
			It("returns the file path to the subdirectory", func() {
				singleSubDirPath := "singleSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.Mkdir(filepath.Join(singleSubDirPath, "test"), 0755)

				dirPath, err := FindOnlySubdir(singleSubDirPath, mockFs)
				Expect(err).NotTo(HaveOccurred())
				Expect(dirPath).To(Equal(filepath.Join(singleSubDirPath, "test")))
			})
		})
		Context("Provided a path with multiple subdirectories", func() {
			It("returns an error", func() {
				multipleSubDirPath := "multipleSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.Mkdir(filepath.Join(multipleSubDirPath, "test"), 0755)
				mockFs.Mkdir(filepath.Join(multipleSubDirPath, "test2"), 0755)

				_, err := FindOnlySubdir(multipleSubDirPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with a no subdirectories", func() {
			It("returns an error", func() {
				noSubDirPath := "noSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.Mkdir(filepath.Join(noSubDirPath), 0755)

				_, err := FindOnlySubdir(noSubDirPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with only files and no subdirectories", func() {
			It("returns the file path to the subdirectory", func() {
				onlyFilesPath := "onlyFilesPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.Mkdir(filepath.Join(onlyFilesPath), 0755)
				mockFs.WriteFile(filepath.Join(onlyFilesPath, "test"), []byte("hi"), 0644)
				mockFs.WriteFile(filepath.Join(onlyFilesPath, "test2"), []byte("bye"), 0644)

				_, err := FindOnlySubdir(onlyFilesPath, mockFs)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Provided a path with files and a single subdirectory", func() {
			It("returns the file path to the subdirectory", func() {
				filesAndSubDirPath := "filesAndSubDirPath"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.MkdirAll(filepath.Join(filesAndSubDirPath, "testSubDir"), 0755)
				mockFs.WriteFile(filepath.Join(filesAndSubDirPath, "test"), []byte("hi"), 0644)
				mockFs.WriteFile(filepath.Join(filesAndSubDirPath, "test2"), []byte("bye"), 0644)

				dirPath, _ := FindOnlySubdir(filesAndSubDirPath, mockFs)
				Expect(dirPath).To(Equal(filepath.Join(filesAndSubDirPath, "testSubDir")))
			})
		})
	})
})
