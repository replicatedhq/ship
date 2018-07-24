// Thanks Gin!
// https://github.com/gin-gonic/contrib/blob/master/static/example/bindata/example.go
package daemon

import (
	"net/http"
	"strings"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type binaryFileSystem struct {
	fs     http.FileSystem
	Logger log.Logger
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	level.Debug(b.Logger).Log("event", "file.open", "name", name)
	return b.fs.Open(name)
}

func (b *binaryFileSystem) Exists(prefix string, filepath string) bool {
	debug := level.Debug(log.With(b.Logger, "prefix", prefix, "filepath", filepath))
	debug.Log("event", "file.exists")
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := b.fs.Open(p); err != nil {
			debug.Log("event", "file.open.err", "err", err)
			return false
		}
		debug.Log("event", "file.open.sucess")
		return true
	}
	debug.Log("event", "file.prefix.miss")
	return false
}

type WebUIBuilder func(root string) *binaryFileSystem

func WebUIFactoryFactory(
	logger log.Logger,
) WebUIBuilder {

	return func(root string) *binaryFileSystem {
		fs := &assetfs.AssetFS{
			Asset:     Asset,
			AssetDir:  AssetDir,
			AssetInfo: AssetInfo,
			Prefix:    root,
		}
		return &binaryFileSystem{
			fs:     fs,
			Logger: logger,
		}
	}
}
