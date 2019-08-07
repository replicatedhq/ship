package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/constants"
)

func Test_readUpstreamFiles(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name                 string
		wantErr              bool
		inputFiles           []fileStruct
		wantUpstreamContents UpstreamContents
	}{
		{
			name: "basic",
			inputFiles: []fileStruct{
				{
					name: "abc.test",
					data: "abc.test",
				},
			},
			wantUpstreamContents: UpstreamContents{
				UpstreamFiles: []UpstreamFile{
					{
						FilePath:     "abc.test",
						FileContents: "YWJjLnRlc3Q=",
					},
				},
				AppRelease: nil,
			},
		},
		{
			name: "app release",
			inputFiles: []fileStruct{
				{
					name: "abc.test",
					data: "abc.test",
				},
				{
					name: "appRelease.json",
					data: `{"id":"appRelease.json"}`,
				},
			},
			wantUpstreamContents: UpstreamContents{
				UpstreamFiles: nil,
				AppRelease:    &ShipRelease{ID: "appRelease.json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			req.NoError(mockFs.MkdirAll(constants.UpstreamContentsPath, os.FileMode(0644)))
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.WriteFile(filepath.Join(constants.UpstreamContentsPath, inFile.name), []byte(inFile.data), os.FileMode(0644)))
			}

			got, err := readUpstreamFiles(mockFs, State{})
			if (err != nil) != tt.wantErr {
				t.Errorf("readUpstreamFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			wantState := State{V1: &V1{UpstreamContents: &tt.wantUpstreamContents}}
			req.Equal(wantState, got)
		})
	}
}

func Test_UpstreamFilesCycle(t *testing.T) {

	tests := []struct {
		name     string
		contents UpstreamContents
	}{
		{
			name: "basic",
			contents: UpstreamContents{
				UpstreamFiles: []UpstreamFile{
					{
						FilePath:     "abc.test",
						FileContents: "YWJjLnRlc3Q=",
					},
				},
				AppRelease: nil,
			},
		},
		{
			name: "app release",
			contents: UpstreamContents{
				UpstreamFiles: nil,
				AppRelease:    &ShipRelease{ID: "appRelease.json"},
			},
		},
		{
			name: "multiple files",
			contents: UpstreamContents{
				UpstreamFiles: []UpstreamFile{
					{
						FilePath:     "abc.test",
						FileContents: "YWJjLnRlc3Q=",
					},
					{
						FilePath:     "subdir/abc.test",
						FileContents: "YWJjLnRlc3Q=",
					},
				},
				AppRelease: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			req.NoError(mockFs.MkdirAll(constants.UpstreamContentsPath, os.FileMode(0644)))

			wantState := State{V1: &V1{UpstreamContents: &tt.contents}}

			err := writeUpstreamFiles(mockFs, wantState)
			req.NoError(err)

			got, err := readUpstreamFiles(mockFs, State{})
			req.NoError(err)

			req.Equal(wantState, got)
		})
	}
}
