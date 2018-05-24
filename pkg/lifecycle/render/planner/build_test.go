package planner

import (
	"errors"
	"testing"

	"context"

	"github.com/golang/mock/gomock"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	docker2 "github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedcom/ship/pkg/test-mocks/config"
	"github.com/replicatedcom/ship/pkg/test-mocks/docker"
	"github.com/replicatedcom/ship/pkg/test-mocks/logger"
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
			name: "registry succeeds",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: nil,
			Expect:                  nil,
		},
		{
			name: "registry succeeds",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: errors.New("nope nope nope"),
			Expect:                  errors.New("nope nope nope"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			v := viper.New()
			daemon := config.NewMockDaemon(mc)
			saver := docker.NewMockImageSaver(mc)
			urlResolver := docker.NewMockPullURLResolver(mc)
			testLogger := &logger.TestLogger{T: t}
			ctx := context.Background()

			pln := &CLIPlanner{
				Logger:      testLogger,
				Fs:          afero.Afero{Fs: afero.NewMemMapFs()},
				UI:          cli.NewMockUi(),
				Viper:       v,
				Daemon:      daemon,
				Saver:       saver,
				URLResolver: urlResolver,
			}

			asset := &api.DockerAsset{
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

			registrySecretSaveOpts := docker2.SaveOpts{
				PullUrl:   "some-pull-url",
				SaveUrl:   asset.Image,
				IsPrivate: asset.Source != "public" && asset.Source != "",
				Filename:  asset.Dest,
				Username:  "tanker",
				Password:  "lutz",
			}

			// When
			step := pln.dockerStep(asset, metadata)

			registrySaveCh := make(chan interface{})
			go func() {
				registrySaveCh <- test.RegistrySecretSaveError
			}()
			saver.EXPECT().SaveImage(ctx, registrySecretSaveOpts).Return(registrySaveCh)

			installIdSaveCh := make(chan interface{})
			go func() {
				installIdSaveCh <- test.InstallationIDSaveError
			}()

			installationIDSaveOpts := docker2.SaveOpts{
				PullUrl:   "some-pull-url",
				SaveUrl:   asset.Image,
				IsPrivate: asset.Source != "public" && asset.Source != "",
				Filename:  asset.Dest,
				Username:  "tanker",
				Password:  "vernon",
			}
			if test.RegistrySecretSaveError == nil {
				saver.EXPECT().SaveImage(ctx, installationIDSaveOpts).Return(installIdSaveCh)
			}

			req := require.New(t)

			err := step.Execute(ctx)
			if test.Expect == nil {
				req.NoError(err)
				return
			}
			req.Equal(test.Expect.Error(), err.Error())

		})
	}
}
