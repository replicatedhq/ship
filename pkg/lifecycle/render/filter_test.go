package render

import (
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func Test_removeDests(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name       string
		inputFiles []fileStruct
		dests      []string
		wantFiles  []fileStruct
		wantErr    bool
	}{
		{
			name: "nonexistent dest",
			inputFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
			dests: []string{"notexist.abc"},
			wantFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
		},
		{
			name: "single file",
			inputFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
				{
					name: "abc/test.file",
					data: "this is a test file",
				},
			},
			dests:     []string{"abc/test.file"},
			wantFiles: []fileStruct{},
		},
		{
			name: "single dir",
			inputFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
				{
					name: "abc/test/a.file",
					data: "this is a test file",
				},
			},
			dests: []string{"abc/test"},
			wantFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
		},
		{
			name: "dir and file",
			inputFiles: []fileStruct{
				{
					name: "another/test.test",
					data: "this is a test file",
				},
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
				{
					name: "abc/test/a.file",
					data: "this is a test file",
				},
				{
					name: "abc/test/b.file",
					data: "this is a test file",
				},
			},
			dests: []string{"abc/test", "another/test.test"},
			wantFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
		},
		{
			name: "illegal absolute dest",
			inputFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
			dests: []string{"/illegal/absolute/path.abc"},
			wantFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
		},
		{
			name: "illegal relative dest",
			inputFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
			dests: []string{"../../../illegal/relative/path.abc"},
			wantFiles: []fileStruct{
				{
					name: "abc/xyz.test",
					data: "this is a test file",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			req.NoError(mockFs.MkdirAll("", os.FileMode(0644)))
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			if err := removeDests(&mockFs, tt.dests); (err != nil) != tt.wantErr {
				t.Errorf("removeDests() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			var expectedFileNames, actualFileNames []string
			for _, expectedFile := range tt.wantFiles {
				expectedFileNames = append(expectedFileNames, expectedFile.name)
			}
			err := mockFs.Walk("", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return errors.Wrapf(err, "walk path %s", path)
				}
				if !info.IsDir() {
					actualFileNames = append(actualFileNames, path)
				}
				return nil
			})
			req.NoError(err)

			req.ElementsMatch(expectedFileNames, actualFileNames, "comparing expected and actual output files, expected %+v got %+v", expectedFileNames, actualFileNames)

			for _, outFile := range tt.wantFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)

				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}
		})
	}
}
