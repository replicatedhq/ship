package config

import (
	"context"
	"net/http"
	"testing"

	"encoding/json"
	"io/ioutil"

	"fmt"
	"math/rand"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
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
			v := viper.New()

			port := rand.Intn(2000) + 33000
			viper.Set("api-port", port)
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			log := &logger.TestLogger{T: t}
			daemon := &ShipDaemon{
				Logger: log,
				Fs:     fs,
				Viper:  v,

				UI: cli.NewMockUi(),
			}

			daemonCtx, daemonCancelFunc := context.WithCancel(context.Background())
			defer daemonCancelFunc()

			errChan := make(chan error)
			log.Log("starting daemon")
			go func() {
				err := daemon.Serve(daemonCtx, test.release)
				log.Log("daemon.error", err)
				errChan <- err
			}()

			// sigh. Give the server a second to start up
			time.Sleep(500 * time.Millisecond)

			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/channel", port))
			require.NoError(t, err)

			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			log.Log("received body", fmt.Sprintf("\"%s\"", body))

			unmarshalled := make(map[string]string)
			err = json.Unmarshal(body, &unmarshalled)
			require.NoError(t, err)

			require.Equal(t, test.expectName, unmarshalled["channelName"])
			require.Equal(t, test.expectIcon, unmarshalled["channelIcon"])
		})
	}
}
