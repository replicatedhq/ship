package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/mitchellh/cli"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type daemonAPITestCase struct {
	name string
	test func(t *testing.T)
}

func initTestDaemon(
	t *testing.T,
	release *api.Release,
	v2 *NavcycleRoutes,
) (*ShipDaemon, int, context.CancelFunc, error) {
	v := viper.New()

	port := rand.Intn(10000) + 33000
	viper.Set("api-port", port)
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	log := &logger.TestLogger{T: t}

	v1 := &V1Routes{
		Logger:           log,
		Fs:               fs,
		Viper:            v,
		UI:               cli.NewMockUi(),
		MessageConfirmed: make(chan string, 1),
		OpenWebConsole:   func(ui cli.Ui, s string, b bool) error { return nil },
	}

	if v2 != nil {
		v.Set("navcycle", true)
		v2.Kustomizer = &kustomize.Kustomizer{}
	}
	daemon := &ShipDaemon{
		Logger:         log,
		WebUIFactory:   WebUIFactoryFactory(log),
		Viper:          v,
		V1Routes:       v1,
		NavcycleRoutes: v2,
	}

	daemonCtx, daemonCancelFunc := context.WithCancel(context.Background())

	log.Log("starting daemon")
	go func() {
		_ = daemon.Serve(daemonCtx, release)
	}()

	var daemonError error
	for i := 0; i < 3; i++ {
		time.Sleep(1 * time.Second)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
		if err == nil && resp.StatusCode == http.StatusOK {
			daemonError = nil
			resp.Body.Close()
			break
		}
		daemonError = err
	}

	if daemonError != nil {
		daemonCancelFunc()
	}
	return daemon, port, daemonCancelFunc, daemonError
}

func TestDaemonAPI(t *testing.T) {
	step1 := api.Step{
		Message: &api.Message{
			Contents: "hello ship!",
			Level:    "info",
		},
		Render: &api.Render{},
	}

	step2 := api.Step{
		Message: &api.Message{
			Contents: "bye ship!",
			Level:    "warn",
		},
		Render: &api.Render{},
	}

	release := &api.Release{
		Spec: api.Spec{
			Lifecycle: api.Lifecycle{
				V1: []api.Step{step1, step2},
			},
			Config: api.Config{
				V1: []libyaml.ConfigGroup{},
			},
		},
	}

	daemon, port, daemonCancelFunc, err := initTestDaemon(t, release, &NavcycleRoutes{Shutdown: make(chan interface{})})
	defer daemonCancelFunc()
	require.New(t).NoError(err)

	tests := []daemonAPITestCase{
		{
			name: "read message before steps",
			test: func(t *testing.T) {
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/message/get", port))
				require.New(t).NoError(err)
				require.New(t).Equal(http.StatusBadRequest, resp.StatusCode)
				bodyStr, err := ioutil.ReadAll(resp.Body)
				require.New(t).NoError(err)
				respMsg := struct {
					Error string `json:"error"`
				}{}
				require.New(t).NoError(json.Unmarshal(bodyStr, &respMsg))
				require.New(t).Equal("no steps taken", respMsg.Error)
			},
		},

		{
			name: "read message after 1st step",
			test: func(t *testing.T) {
				daemon.PushMessageStep(context.Background(), daemontypes.Message{
					Contents: step1.Message.Contents,
					Level:    step1.Message.Level,
				}, MessageActions())

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/message/get", port))
				require.New(t).NoError(err)
				require.New(t).Equal(http.StatusOK, resp.StatusCode)
				bodyStr, err := ioutil.ReadAll(resp.Body)
				require.New(t).NoError(err)
				respMsg := struct {
					Message api.Message `json:"message"`
				}{}
				require.New(t).NoError(json.Unmarshal(bodyStr, &respMsg))
				require.New(t).Equal(step1.Message, &respMsg.Message)
			},
		},

		{
			name: "confirm message that is not current",
			test: func(t *testing.T) {
				log := &logger.TestLogger{T: t}
				daemon.PushMessageStep(context.Background(), daemontypes.Message{
					Contents: step2.Message.Contents,
					Level:    step2.Message.Level,
				}, MessageActions())

				reqBody := bytes.NewReader([]byte(`{"step_name": "wrong-name"}`))
				log.Log("daemon.current", daemon.currentStepName)
				resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/v1/message/confirm", port), "application/json", reqBody)
				require.New(t).NoError(err)
				require.New(t).Equal(http.StatusBadRequest, resp.StatusCode)
				bodyStr, err := ioutil.ReadAll(resp.Body)
				require.New(t).NoError(err)
				respMsg := struct {
					Error string `json:"error"`
				}{}
				require.New(t).NoError(json.Unmarshal(bodyStr, &respMsg))
				require.New(t).Equal("not current step", respMsg.Error)
			},
		},

		{
			name: "confirm message that is current",
			test: func(t *testing.T) {
				log := &logger.TestLogger{T: t}
				reqBody := bytes.NewReader([]byte(`{"step_name": "message"}`))
				log.Log("daemon.current", daemon.currentStepName)
				resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/v1/message/confirm", port), "application/json", reqBody)
				require.New(t).NoError(err)
				require.New(t).Equal(http.StatusOK, resp.StatusCode)
				msg := <-daemon.MessageConfirmedChan()
				require.New(t).Equal("message", msg)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.test(t)
		})
	}
}
