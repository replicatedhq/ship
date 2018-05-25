package state

import (
	"encoding/json"
	"path/filepath"

	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/fs"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// StateManager is the saved output of a plan run to load on future runs
type StateManager struct {
	Logger log.Logger
	FS     afero.Afero
}

func ManagerFromViper(v *viper.Viper) *StateManager {
	return &StateManager{
		Logger: logger.FromViper(v),
		FS:     fs.FromViper(v),
	}
}

// Serialize takes the application data and input params and serializes a state file to disk
func (s StateManager) Serialize(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	serialized, err := json.Marshal(templateContext)
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	if err = s.FS.MkdirAll(filepath.Dir(Path), 0700); err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = s.FS.WriteFile(Path, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}

// TryLoad will attempt to load a state file from disk, if present
func (s *StateManager) TryLoad() (map[string]interface{}, error) {
	if _, err := s.FS.Stat(Path); os.IsNotExist(err) {
		level.Debug(s.Logger).Log("msg", "no saved state exists", "path", Path)
		return make(map[string]interface{}), nil
	}

	serialized, err := s.FS.ReadFile(Path)
	if err != nil {
		return nil, errors.Wrap(err, "read state file")
	}

	templateContext := make(map[string]interface{})
	if err := json.Unmarshal(serialized, &templateContext); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	return templateContext, nil
}
