package docker

import (
	"errors"
	"testing"

	"github.com/replicatedhq/libyaml"

	"context"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/images"
	mockimages "github.com/replicatedhq/ship/pkg/test-mocks/images"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestDockerStep(t *testing.T) {
	tests := []struct {
		name                    string
		RegistrySecretSaveError error
		InstallationIDSaveError error
		Expect                  error
	}{
		{
			name: "registry succeeds",
			RegistrySecretSaveError: nil,
			InstallationIDSaveError: nil,
			Expect:                  nil,
		},
		{
			name: "registry fails, install id succeeds",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: nil,
			Expect:                  nil,
		},
		{
			name: "registry fails, install id fails",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: errors.New("nope nope nope"),
			Expect:                  errors.New("docker save image, both auth methods failed: nope nope nope"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			v := viper.New()
			saver := mockimages.NewMockImageSaver(mc)
			urlResolver := mockimages.NewMockPullURLResolver(mc)
			testLogger := &logger.TestLogger{T: t}
			ctx := context.Background()

			step := &DefaultStep{
				Logger:      testLogger,
				Fs:          afero.Afero{Fs: afero.NewMemMapFs()},
				URLResolver: urlResolver,
				ImageSaver:  saver,
				Viper:       v,
			}

			asset := api.DockerAsset{
				Image:  "registry.replicated.com/retracedio/api:v2.0.0",
				Source: "replicated",
			}
			metadata := api.ReleaseMetadata{
				CustomerID:     "tanker",
				RegistrySecret: "lutz",
				Images:         []api.Image{},
			}
			v.Set("installation-id", "vernon")

			urlResolver.EXPECT().ResolvePullURL(asset, metadata).Return("some-pull-url", nil)

			registrySecretSaveOpts := images.SaveOpts{
				PullURL:   "some-pull-url",
				SaveURL:   asset.Image,
				IsPrivate: asset.Source != "public" && asset.Source != "",
				Filename:  asset.Dest,
				Username:  "tanker",
				Password:  "lutz",
			}

			registrySaveCh := make(chan interface{})
			go func() {
				registrySaveCh <- test.RegistrySecretSaveError
				close(registrySaveCh)
			}()
			saver.EXPECT().SaveImage(ctx, registrySecretSaveOpts).Return(registrySaveCh)

			if test.RegistrySecretSaveError != nil {
				installIDSaveCh := make(chan interface{})
				go func() {
					installIDSaveCh <- test.InstallationIDSaveError
					close(installIDSaveCh)
				}()

				installationIDSaveOpts := images.SaveOpts{
					PullURL:   "some-pull-url",
					SaveURL:   asset.Image,
					IsPrivate: asset.Source != "public" && asset.Source != "",
					Filename:  asset.Dest,
					Username:  "tanker",
					Password:  "vernon",
				}
				saver.EXPECT().SaveImage(ctx, installationIDSaveOpts).Return(installIDSaveCh)
			}

			req := require.New(t)

			// When
			err := step.Execute(asset, metadata, mockProgress, asset.Dest, map[string]interface{}{}, []libyaml.ConfigGroup{})(ctx)

			// Then
			if test.Expect == nil {
				req.NoError(err)
				return
			}
			if err == nil {
				req.FailNowf("expected error did not occur", "expected error \"%v\" to be returned by step", test.Expect)
			}

			req.Equal(test.Expect.Error(), err.Error(), "expected errors to be equal")

		})
	}
}

func mockProgress(ch chan interface{}, debug log.Logger) error {
	var saveError error
	for msg := range ch {
		if msg == nil {
			continue
		}
		switch v := msg.(type) {
		case error:
			// continue reading on error to ensure channel is not blocked
			saveError = v
			debug.Log("event", "error", "message", fmt.Sprintf("%#v", v))
		default:
			debug.Log("event", "progress", "message", fmt.Sprintf("%#v", v))
		}
	}
	return saveError
}
