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

type Manager interface {
	SerializeHelmValues(values string) error
	SerializeConfig(
		assets []api.Asset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
	) error
	TryLoad() (State, error)
	RemoveStateFile() error
	SaveKustomize(kustomize *Kustomize) error
}

var _ Manager = &MManager{}

// MManager is the saved output of a plan run to load on future runs
type MManager struct {
	Logger log.Logger
	FS     afero.Afero
	V      *viper.Viper
}

func NewManager(
	logger log.Logger,
	fs afero.Afero,
	v *viper.Viper,
) Manager {
	return &MManager{
		Logger: logger,
		FS:     fs,
		V:      v,
	}
}

// SerializeHelmValues takes user input helm values and serializes a state file to disk
func (s *MManager) SerializeHelmValues(values string) error {
	debug := level.Debug(log.With(s.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "tryLoadState")
	currentState, err := s.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.HelmValues = values

	return s.serializeAndWriteState(versionedState)
}

// SerializeConfig takes the application data and input params and serializes a state file to disk
func (s *MManager) SerializeConfig(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	debug := level.Debug(log.With(s.Logger, "method", "serializeConfig"))

	debug.Log("event", "tryLoadState")
	currentState, err := s.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.Config = templateContext

	return s.serializeAndWriteState(versionedState)
}

// TryLoad will attempt to load a state file from disk, if present
func (s *MManager) TryLoad() (State, error) {
	statePath := s.V.GetString("state-file")
	if statePath == "" {
		statePath = constants.StatePath
	}

	if _, err := s.FS.Stat(statePath); os.IsNotExist(err) {
		level.Debug(s.Logger).Log("msg", "no saved state exists", "path", statePath)
		return Empty{}, nil
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

	if state.V1 != nil {
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

func (m *MManager) SaveKustomize(kustomize *Kustomize) error {
	currentState, err := m.TryLoad()
	if err != nil {
		return errors.Wrapf(err, "load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.Kustomize = kustomize

	if err := m.serializeAndWriteState(versionedState); err != nil {
		return errors.Wrap(err, "write state")
	}

	return nil
}

// RemoveStateFile will attempt to remove the state file from disk
func (m *MManager) RemoveStateFile() error {
	statePath := m.V.GetString("state-file")
	if statePath == "" {
		statePath = constants.StatePath
	}

	err := m.FS.Remove(statePath)
	if err != nil {
		return errors.Wrap(err, "remove state file")
	}

	return nil
}

func (s *MManager) serializeAndWriteState(state VersionedState) error {
	state.V1.ChartURL = state.CurrentChartURL() // chart URL persists throughout `init` lifecycle

	serialized, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	err = s.FS.MkdirAll(filepath.Dir(constants.StatePath), 0700)
	if err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = s.FS.WriteFile(constants.StatePath, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}
