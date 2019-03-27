package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type daemonChannelTest struct {
	name       string
	release    *api.Release
	expectName string
	expectIcon string
}

func TestDaemonChannel(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	tests := []daemonChannelTest{
		{
			name: "test_channel_noicon",
			release: &api.Release{
				Metadata: api.ReleaseMetadata{
					Type:        "replicated.app",
					ChannelName: "Application",
					ChannelIcon: "",
				},
			},
			expectName: "Application",
			expectIcon: "",
		},
		{
			name: "test_channel_withicon",
			release: &api.Release{
				Metadata: api.ReleaseMetadata{
					Type:        "replicated.app",
					ChannelName: "Clubhouse Enterprise",
					ChannelIcon: "https://frontend-production-cdn.clubhouse.io/v0.5.20180509155736/images/logos/clubhouse_mascot_180x180.png",
				},
			},
			expectName: "Clubhouse Enterprise",
			expectIcon: "https://frontend-production-cdn.clubhouse.io/v0.5.20180509155736/images/logos/clubhouse_mascot_180x180.png",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			v := viper.New()

			port := rand.Intn(2000) + 33000
			viper.Set("api-port", port)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			log := &logger.TestLogger{T: t}
			daemon := &ShipDaemon{
				Logger:       log,
				WebUIFactory: WebUIFactoryFactory(log),
				Viper:        v,
				V1Routes: &V1Routes{
					Logger: log,
					Fs:     fs,
					Viper:  v,

					UI:             cli.NewMockUi(),
					OpenWebConsole: func(ui cli.Ui, s string, b bool) error { return nil },
				},
				NavcycleRoutes: &NavcycleRoutes{
					Kustomizer: &kustomize.Kustomizer{},
					Shutdown:   make(chan interface{}),
				},
			}

			daemonCtx, daemonCancelFunc := context.WithCancel(context.Background())

			errChan := make(chan error)
			log.Log("starting daemon")
			go func(errCh chan error) {
				err := daemon.Serve(daemonCtx, test.release)
				log.Log("daemon.error", err)
				errCh <- err
			}(errChan)

			// sigh. Give the server a second to start up
			time.Sleep(500 * time.Millisecond)

			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/metadata", port))
			req.NoError(err)

			body, err := ioutil.ReadAll(resp.Body)
			req.NoError(err)
			log.Log("received body", fmt.Sprintf("\"%s\"", body))

			daemonCancelFunc()

			unmarshalled := make(map[string]string)
			err = json.Unmarshal(body, &unmarshalled)
			req.NoError(err)

			req.Equal(test.expectName, unmarshalled["name"])
			req.Equal(test.expectIcon, unmarshalled["icon"])

			daemonErr := <-errChan
			req.EqualError(daemonErr, "context canceled")
		})
	}
}
