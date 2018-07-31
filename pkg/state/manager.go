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

// SerializeHelmValues takes user input helm values and serializes a state file to disk
func (s *Manager) SerializeHelmValues(values string) error {
	debug := level.Debug(log.With(s.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "tryLoadState")
	currentState, err := s.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}

	debug.Log("event", "emptyState")
	isEmpty := currentState == Empty{}
	if isEmpty {
		toSerialize := VersionedState{V1: &V1{HelmValues: values}}
		return s.serializeAndWriteState(toSerialize)
	}

	debug.Log("event", "serializeAndWriteState", "change", "helmValues")
	toSerialize := currentState.(VersionedState)
	toSerialize.V1.HelmValues = values
	return s.serializeAndWriteState(toSerialize)
}

// SerializeChartURL takes the URL of the helm chart and serializes a state file to disk
func (s *Manager) SerializeChartURL(URL string) error {
	debug := level.Debug(log.With(s.Logger, "method", "SerializeChartURL"))

	debug.Log("event", "tryLoadState")
	currentState, err := s.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}

	debug.Log("event", "emptyState")
	isEmpty := currentState == Empty{}
	if isEmpty {
		toSerialize := VersionedState{V1: &V1{ChartURL: URL}}
		return s.serializeAndWriteState(toSerialize)
	}

	debug.Log("event", "serializeAndWriteState", "change", "helmChartURL")
	toSerialize := currentState.(VersionedState)
	toSerialize.V1.ChartURL = URL
	return s.serializeAndWriteState(toSerialize)
}

// Serialize takes the application data and input params and serializes a state file to disk
func (s *Manager) Serialize(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	toSerialize := VersionedState{V1: &V1{Config: templateContext}}
	return s.serializeAndWriteState(toSerialize)
}

func (s *Manager) serializeAndWriteState(state VersionedState) error {
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

// TryLoad will attempt to load a state file from disk, if present
func (s *Manager) TryLoad() (State, error) {
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

func (m *Manager) SaveKustomize(kustomize *Kustomize) error {
	state, err := m.TryLoad()
	if err != nil {
		return errors.Wrapf(err, "load state")
	}

	newState := VersionedState{
		V1: &V1{
			Config:    state.CurrentConfig(),
			Kustomize: kustomize,
		},
	}

	if err := m.serializeAndWriteState(newState); err != nil {
		return errors.Wrap(err, "write state")
	}

	return nil
}

// RemoveStateFile will attempt to remove the state file from disk
func (m *Manager) RemoveStateFile() error {
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
