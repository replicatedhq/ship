package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Manager is the saved output of a plan run to load on future runs
type Manager struct {
	Logger log.Logger
	FS     afero.Afero
	V      *viper.Viper
}

func NewManager(
	logger log.Logger,
	fs afero.Afero,
	v *viper.Viper,
) *Manager {
	return &Manager{
		Logger: logger,
		FS:     fs,
		V:      v,
	}
}

type State interface {
	CurrentConfig() map[string]interface{}
}

type V0 map[string]interface{}
type V1 struct {
	Config    map[string]interface{} `json:"config" yaml:"config" hcl:"config"`
	Terraform interface{}            `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
}

var _ State = VersionedState{}
var _ State = empty{}
var _ State = V0{}

type VersionedState struct {
	V1 *V1 `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

func (u VersionedState) CurrentConfig() map[string]interface{} {
	if u.V1 != nil && u.V1.Config != nil {
		return u.V1.Config
	}
	return make(map[string]interface{})
}

type empty struct{}

func (empty) CurrentConfig() map[string]interface{} {
	return make(map[string]interface{})
}

func (v V0) CurrentConfig() map[string]interface{} {
	return v
}

// Serialize takes the application data and input params and serializes a state file to disk
func (s Manager) Serialize(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	toSerialize := VersionedState{V1: &V1{Config: templateContext}}
	serialized, err := json.Marshal(toSerialize)
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	if err = s.FS.MkdirAll(filepath.Dir(constants.StatePath), 0700); err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = s.FS.WriteFile(constants.StatePath, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}

// TryLoad will attempt to load a state file from disk, if present
func (s *Manager) TryLoad() (State, error) {
	statePath := s.V.GetString("state-file")
	if statePath == "" {
		statePath = constants.StatePath
	}

	if _, err := s.FS.Stat(statePath); os.IsNotExist(err) {
		level.Debug(s.Logger).Log("msg", "no saved state exists", "path", statePath)
		return empty{}, nil
	}

	serialized, err := s.FS.ReadFile(statePath)
	if err != nil {
		return nil, errors.Wrap(err, "read state file")
	}

	// HACK -- try to deserialize it as VersionedState, otherwise, assume its a raw map of config values
	var state VersionedState
	if err := json.Unmarshal(serialized, &state); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(s.Logger).Log("event", "state.unmarshal", "type", "versioned", "value", state)

	if state.V1 != nil && state.V1.Config != nil {
		level.Debug(s.Logger).Log("event", "state.resolve", "type", "versioned")
		return state, nil
	}

	var mapState map[string]interface{}
	if err := json.Unmarshal(serialized, &mapState); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(s.Logger).Log("event", "state.resolve", "type", "raw")
	return V0(mapState), nil
}
