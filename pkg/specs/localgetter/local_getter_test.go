package localgetter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestLocalGetter_copyDir(t *testing.T) {
	type file struct {
		contents []byte
		path     string
	}
	tests := []struct {
		name     string
		upstream string
		savePath string
		inFiles  []file
		outFiles []file
		wantErr  bool
	}{
		{
			name:     "single file",
			upstream: "/upstream/file",
			savePath: "/save/file",
			inFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/file",
				},
			},
			outFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/file",
				},
				{
					contents: []byte("hello world"),
					path:     "/save/file",
				},
			},
		},
		{
			name:     "single file in dir",
			upstream: "/upstream/dir",
			savePath: "/save/dir",
			inFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/dir/file",
				},
			},
			outFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/dir/file",
				},
				{
					contents: []byte("hello world"),
					path:     "/save/dir/file",
				},
			},
		},
		{
			name:     "file plus subdirs",
			upstream: "/upstream/",
			savePath: "/save/",
			inFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/dir/file",
				},
				{
					contents: []byte("abc xyz"),
					path:     "/upstream/dir2/file",
				},
				{
					contents: []byte("123456789"),
					path:     "/upstream/file",
				},
			},
			outFiles: []file{
				{
					contents: []byte("hello world"),
					path:     "/upstream/dir/file",
				},
				{
					contents: []byte("abc xyz"),
					path:     "/upstream/dir2/file",
				},
				{
					contents: []byte("123456789"),
					path:     "/upstream/file",
				},
				{
					contents: []byte("hello world"),
					path:     "/save/dir/file",
				},
				{
					contents: []byte("abc xyz"),
					path:     "/save/dir2/file",
				},
				{
					contents: []byte("123456789"),
					path:     "/save/file",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			mmFs := afero.Afero{Fs: afero.NewMemMapFs()}

			g := &LocalGetter{
				Logger: log.NewNopLogger(),
				FS:     mmFs,
			}

			for _, file := range tt.inFiles {
				req.NoError(mmFs.MkdirAll(filepath.Dir(file.path), os.ModePerm))
				req.NoError(mmFs.WriteFile(file.path, file.contents, os.ModePerm))
			}

			err := g.copyDir(context.Background(), tt.upstream, tt.savePath)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			for _, file := range tt.outFiles {
				contents, err := mmFs.ReadFile(file.path)
				req.NoError(err)
				req.Equal(file.contents, contents, "expected equal contents: expected %q, got %q", string(file.contents), string(contents))
			}
		})
	}
}
