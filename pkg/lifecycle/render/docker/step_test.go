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
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	mockimages "github.com/replicatedhq/ship/pkg/test-mocks/images"
	mocksaver "github.com/replicatedhq/ship/pkg/test-mocks/images/saver"
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
			name:                    "registry succeeds",
			RegistrySecretSaveError: nil,
			InstallationIDSaveError: nil,
			Expect:                  nil,
		},
		{
			name:                    "registry fails, install id succeeds",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: nil,
			Expect:                  nil,
		},
		{
			name:                    "registry fails, install id fails",
			RegistrySecretSaveError: errors.New("noooope"),
			InstallationIDSaveError: errors.New("nope nope nope"),
			Expect:                  errors.New("docker save image, both auth methods failed: nope nope nope"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			v := viper.New()
			saver := mocksaver.NewMockImageSaver(mc)
			urlResolver := mockimages.NewMockPullURLResolver(mc)
			testLogger := &logger.TestLogger{T: t}
			bb := templates.NewBuilderBuilder(testLogger, v)
			ctx := context.Background()

			step := &DefaultStep{
				Logger:         testLogger,
				URLResolver:    urlResolver,
				ImageSaver:     saver,
				Viper:          v,
				BuilderBuilder: bb,
			}

			asset := api.DockerAsset{
				AssetShared: api.AssetShared{
					Dest: "{{repl ConfigOption \"docker_dir\" }}/image.tar",
				},
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

			templateContext := map[string]interface{}{
				"docker_dir": "images",
			}
			configGroups := []libyaml.ConfigGroup{
				{
					Name: "Test",
					Items: []*libyaml.ConfigItem{
						{
							Name: "docker_dir",
							Type: "text",
						},
					},
				},
			}

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
			err := step.Execute(
				root.Fs{
					Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
					RootPath: "",
				},
				asset,
				metadata,
				mockProgress,
				asset.Dest,
				templateContext,
				configGroups,
			)(ctx)

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
