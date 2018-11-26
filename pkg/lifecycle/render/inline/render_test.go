package inline

import (
	"context"
	"testing"

	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestInlineRender(t *testing.T) {
	tests := []struct {
		name            string
		asset           api.InlineAsset
		meta            api.ReleaseMetadata
		templateContext map[string]interface{}
		configGroups    []libyaml.ConfigGroup
		expect          map[string]interface{}
	}{
		{
			name: "happy path",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "foo.txt",
				},
			},
			expect: map[string]interface{}{
				"foo.txt": "hello!",
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			bb := templates.NewBuilderBuilder(testLogger, v, &state.MockManager{})
			rootFs := root.Fs{
				Afero:    afero.Afero{Fs: afero.NewMemMapFs()},
				RootPath: "",
			}

			renderer := &LocalRenderer{
				Logger:         testLogger,
				Viper:          v,
				BuilderBuilder: bb,
			}

			err := renderer.Execute(
				rootFs,
				test.asset,
				test.meta,
				test.templateContext,
				test.configGroups,
			)(context.Background())
			req.NoError(err)

			for filename, expectContents := range test.expect {
				contents, err := rootFs.ReadFile(filename)
				req.NoError(err)
				req.Equal(expectContents, string(contents))
			}
		})
	}
}
