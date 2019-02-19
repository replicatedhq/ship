package templates

import (
	"os"
	"testing"

	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestBuildDir(t *testing.T) {
	type file struct {
		contents string
		path     string
	}

	tests := []struct {
		name        string
		buildPath   string
		inputFiles  []file
		outputFiles []file
	}{
		{
			name:      "no templates",
			buildPath: "dir",
			inputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
			},
			outputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
			},
		},
		{
			name:      "template not in dir",
			buildPath: "dir",
			inputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
				{
					contents: `{{repl ConfigOption "option_1"}}`,
					path:     "notDir/template.txt",
				},
			},
			outputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
				{
					contents: `{{repl ConfigOption "option_1"}}`,
					path:     "notDir/template.txt",
				},
			},
		},
		{
			name:      "template in dir",
			buildPath: "dir",
			inputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
				{
					contents: `{{repl ConfigOption "option_1"}}`,
					path:     "dir/template.txt",
				},
			},
			outputFiles: []file{
				{
					contents: "notATemplate",
					path:     "dir/file.txt",
				},
				{
					contents: "Option 1",
					path:     "dir/template.txt",
				},
			},
		},
		{
			name:      "template in subdir",
			buildPath: "anotherdir",
			inputFiles: []file{
				{
					contents: "notATemplate",
					path:     "anotherdir/file.txt",
				},
				{
					contents: `{{repl ConfigOption "option_2"}}`,
					path:     "anotherdir/subdir/template.txt",
				},
			},
			outputFiles: []file{
				{
					contents: "notATemplate",
					path:     "anotherdir/file.txt",
				},
				{
					contents: "Option 2",
					path:     "anotherdir/subdir/template.txt",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			for _, file := range tt.inputFiles {
				req.NoError(fs.WriteFile(file.path, []byte(file.contents), os.FileMode(777)))
			}

			builderBuilder := &BuilderBuilder{
				Logger: &logger.TestLogger{T: t},
				Viper:  viper.New(),
			}

			builder := builderBuilder.NewBuilder(
				builderBuilder.NewStaticContext(),
				testContext{},
			)

			err := BuildDir(tt.buildPath, &fs, &builder)
			req.NoError(err)

			for _, file := range tt.outputFiles {
				actualContents, err := fs.ReadFile(file.path)
				req.NoError(err)
				req.Equal(file.contents, string(actualContents))
			}
		})
	}
}
