package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	SerializeChartURL(URL string) error
	SerializeContentSHA(contentSHA string) error
	Save(v VersionedState) error
}

var _ Manager = &MManager{}

// MManager is the saved output of a plan run to load on future runs
type MManager struct {
	Logger log.Logger
	FS     afero.Afero
	V      *viper.Viper
}

func (m *MManager) Save(v VersionedState) error {
	return m.serializeAndWriteState(v)
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

// SerializeChartURL is used by `ship init` to serialize a state file with ChartURL to disk
func (s *MManager) SerializeChartURL(URL string) error {
	debug := level.Debug(log.With(s.Logger, "method", "SerializeChartURL"))

	debug.Log("event", "generateChartURLState")
	toSerialize := VersionedState{V1: &V1{ChartURL: URL}}

	return s.serializeAndWriteState(toSerialize)
}

// SerializeContentSHA writes the contentSHA to the state file
func (m *MManager) SerializeContentSHA(contentSHA string) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeContentSHA"))

	debug.Log("event", "tryLoadState")
	currentState, err := m.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.ContentSHA = contentSHA

	return m.serializeAndWriteState(versionedState)
}

// SerializeHelmValues takes user input helm values and serializes a state file to disk
func (m *MManager) SerializeHelmValues(values string) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "tryLoadState")
	currentState, err := m.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.HelmValues = values

	return m.serializeAndWriteState(versionedState)
}

// SerializeConfig takes the application data and input params and serializes a state file to disk
func (m *MManager) SerializeConfig(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeConfig"))

	debug.Log("event", "tryLoadState")
	currentState, err := m.TryLoad()
	if err != nil {
		return errors.Wrap(err, "try load state")
	}
	versionedState := currentState.Versioned()
	versionedState.V1.Config = templateContext

	return m.serializeAndWriteState(versionedState)
}

// TryLoad will attempt to load a state file from disk, if present
func (m *MManager) TryLoad() (State, error) {
	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" {
		stateFrom = "file"
	}

	switch stateFrom {
	case "file":
		return m.tryLoadFromFile()
	case "secret":
		return m.tryLoadFromSecret()
	default:
		err := fmt.Errorf("unsupported state-from value: %q", stateFrom)
		return nil, errors.Wrap(err, "try load state")
	}
}

// tryLoadFromSecret will attempt to load the state from a secret
// currently only supports in-cluster execution
func (m *MManager) tryLoadFromSecret() (State, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get in cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "get kubernetes client")
	}

	secret, err := clientset.CoreV1().Secrets(m.V.GetString("secret-namespace")).Get(m.V.GetString("secret-name"), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get secret")
	}

	serialized, ok := secret.Data[m.V.GetString("secret-key")]
	if !ok {
		err := fmt.Errorf("key %q not found in secret %q", m.V.GetString("secret-key"), m.V.GetString("secret-name"))
		return nil, errors.Wrap(err, "get state from secret")
	}

	// HACK -- try to deserialize it as VersionedState, otherwise, assume its a raw map of config values
	var state VersionedState
	if err := json.Unmarshal(serialized, &state); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(m.Logger).Log(
		"event", "state.unmarshal",
		"type", "versioned",
		"source", "secret",
		"value", fmt.Sprintf("%+v", state),
	)

	if state.V1 != nil {
		level.Debug(m.Logger).Log("event", "state.resolve", "type", "versioned")
		return state, nil
	}

	var mapState map[string]interface{}
	if err := json.Unmarshal(serialized, &mapState); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(m.Logger).Log("event", "state.resolve", "type", "raw")
	return V0(mapState), nil
}

func (m *MManager) tryLoadFromFile() (State, error) {
	statePath := m.V.GetString("state-file")
	if statePath == "" {
		statePath = constants.StatePath
	}

	if _, err := m.FS.Stat(statePath); os.IsNotExist(err) {
		level.Debug(m.Logger).Log("msg", "no saved state exists", "path", statePath)
		return Empty{}, nil
	}

	serialized, err := m.FS.ReadFile(statePath)
	if err != nil {
		return nil, errors.Wrap(err, "read state file")
	}

	// HACK -- try to deserialize it as VersionedState, otherwise, assume its a raw map of config values
	var state VersionedState
	if err := json.Unmarshal(serialized, &state); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(m.Logger).Log(
		"event", "state.unmarshal",
		"type", "versioned",
		"source", "file",
		"value", fmt.Sprintf("%+v", state),
	)

	if state.V1 != nil {
		level.Debug(m.Logger).Log("event", "state.resolve", "type", "versioned")
		return state, nil
	}

	var mapState map[string]interface{}
	if err := json.Unmarshal(serialized, &mapState); err != nil {
		return nil, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(m.Logger).Log("event", "state.resolve", "type", "raw")
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

func (m *MManager) serializeAndWriteState(state VersionedState) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeHelmValues"))

	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" {
		stateFrom = "file"
	}

	debug.Log("event", "serializeAndWriteState", "stateFrom", stateFrom)

	switch stateFrom {
	case "file":
		return m.serializeAndWriteStateFile(state)
	case "secret":
		return m.serializeAndWriteStateSecret(state)
	default:
		err := fmt.Errorf("unsupported state-from value: %q", stateFrom)
		return errors.Wrap(err, "serializeAndWriteState")
	}
}

func (m *MManager) serializeAndWriteStateFile(state VersionedState) error {
	state.V1.ChartURL = state.CurrentChartURL()

	serialized, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	err = m.FS.MkdirAll(filepath.Dir(constants.StatePath), 0700)
	if err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = m.FS.WriteFile(constants.StatePath, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}

func (m *MManager) serializeAndWriteStateSecret(state VersionedState) error {
	state.V1.ChartURL = state.CurrentChartURL()

	serialized, err := json.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "get in cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "get kubernetes client")
	}

	secret, err := clientset.CoreV1().Secrets(m.V.GetString("secret-namespace")).Get(m.V.GetString("secret-name"), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get secret")
	}

	secret.Data[m.V.GetString("secret-key")] = serialized
	secret.Data["marc"] = []byte("Asdasd")
	debug := level.Debug(log.With(m.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "serializeAndWriteStateSecret", "name", secret.Name, "key", m.V.GetString("secret-key"))

	_, err = clientset.CoreV1().Secrets(m.V.GetString("secret-namespace")).Update(secret)
	if err != nil {
		return errors.Wrap(err, "update secret")
	}

	return nil
}
