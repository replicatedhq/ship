package stategetter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"

	"github.com/spf13/afero"
)

func TestStateGetter_GetFiles(t *testing.T) {
	type fileType struct {
		Path     string
		Contents string
		Perms    os.FileMode
	}

	tests := []struct {
		name            string
		upstream        string
		destinationPath string
		want            string
		wantErr         bool
		contents        state.UpstreamContents
		inputFS         []fileType
		outputFS        []fileType
	}{
		{
			name:            "no contents",
			upstream:        "blah",
			destinationPath: "/hello-world",
			want:            "/hello-world/state",
			contents:        state.UpstreamContents{},
			inputFS:         []fileType{},
			outputFS:        []fileType{},
		},
		{
			name:            "nested contents",
			upstream:        "blah",
			destinationPath: "/hello-world",
			want:            "/hello-world/state",
			contents: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "README.md", FileContents: "SGVsbG8gV29ybGQh"},
					{FilePath: "/nest/me/GOODBYE.md", FileContents: "R29vZGJ5ZQ=="},
				},
			},
			inputFS: []fileType{},
			outputFS: []fileType{
				{
					Path:     "/hello-world/state/README.md",
					Contents: "Hello World!",
					Perms:    0755,
				},
				{
					Path:     "/hello-world/state/nest/me/GOODBYE.md",
					Contents: "Goodbye",
					Perms:    0755,
				},
			},
		},
		{
			name:            "overwrite contents",
			upstream:        "blah",
			destinationPath: "/hello-world",
			want:            "/hello-world/state",
			contents: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "README.md", FileContents: "SGVsbG8gV29ybGQh"},
					{FilePath: "/nest/me/GOODBYE.md", FileContents: "R29vZGJ5ZQ=="},
				},
			},
			inputFS: []fileType{
				{
					Path:     "/hello-world/state/README.md",
					Contents: "Should Not Exist",
					Perms:    0755,
				},
			},
			outputFS: []fileType{
				{
					Path:     "/hello-world/state/README.md",
					Contents: "Hello World!",
					Perms:    0755,
				},
				{
					Path:     "/hello-world/state/nest/me/GOODBYE.md",
					Contents: "Goodbye",
					Perms:    0755,
				},
			},
		},
		{
			name:            "not base64 contents",
			upstream:        "blah",
			destinationPath: "/hello-world",
			want:            "",
			wantErr:         true,
			contents: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "README.md", FileContents: "SGVsbG8gV29ybGQh"},
					{FilePath: "/nest/me/GOODBYE.md", FileContents: "this-is-not-valid-base64"},
				},
			},
			inputFS: []fileType{
				{
					Path:     "/hello-world/state/README.md",
					Contents: "Should Not Exist",
					Perms:    0755,
				},
			},
			outputFS: []fileType{
				{
					Path:     "/hello-world/state/README.md",
					Contents: "Hello World!",
					Perms:    0755,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			tlog := logger.TestLogger{T: t}
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			g := &StateGetter{
				Logger:   &tlog,
				Contents: &tt.contents,
				Fs:       fs,
			}

			// populate input filesystem
			for _, file := range tt.inputFS {
				err := fs.MkdirAll(filepath.Dir(file.Path), file.Perms)
				req.NoError(err)

				err = fs.WriteFile(file.Path, []byte(file.Contents), file.Perms)
				req.NoError(err)
			}

			got, err := g.GetFiles(context.Background(), tt.upstream, tt.destinationPath)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(tt.want, got)

			// compare resulting filesystems
			for _, file := range tt.outputFS {
				info, err := fs.Stat(file.Path)
				req.NoError(err)
				req.Equal(file.Perms, info.Mode())

				contents, err := fs.ReadFile(file.Path)
				req.NoError(err)
				req.Equal(file.Contents, string(contents))
			}
		})
	}
}
