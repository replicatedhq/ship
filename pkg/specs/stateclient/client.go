package stateclient

import (
	"context"
	"encoding/base64"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type StateClient struct {
	Logger   log.Logger
	Contents *state.UpstreamContents
	Fs       afero.Afero
}

func NewStateClient(fs afero.Afero, logger log.Logger, contents *state.UpstreamContents) *StateClient {
	return &StateClient{
		Contents: contents,
		Fs:       fs,
		Logger:   logger,
	}
}

func (g *StateClient) GetFiles(
	ctx context.Context,
	upstream string,
	destinationPath string,
) (string, error) {
	debug := level.Debug(log.With(g.Logger, "method", "getStatefileContents"))

	debug.Log("event", "removeAll", "destinationPath", destinationPath)
	err := g.Fs.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.Wrap(err, "remove state destination")
	}

	stateUnpackPath := filepath.Join(destinationPath, "state")

	for _, upstreamFile := range g.Contents.UpstreamFiles {
		err = g.Fs.MkdirAll(filepath.Join(stateUnpackPath, filepath.Dir(upstreamFile.FilePath)), 0755)
		if err != nil {
			return "", errors.Wrapf(err, "create dir for file %s", upstreamFile.FilePath)
		}

		rawContents, err := base64.StdEncoding.DecodeString(upstreamFile.FileContents)
		if err != nil {
			return "", errors.Wrapf(err, "decode contents of file %s", upstreamFile.FilePath)
		}

		err = g.Fs.WriteFile(filepath.Join(stateUnpackPath, upstreamFile.FilePath), rawContents, 0755)
		if err != nil {
			return "", errors.Wrapf(err, "write contents of file %s", upstreamFile.FilePath)
		}
	}

	return stateUnpackPath, nil
}
