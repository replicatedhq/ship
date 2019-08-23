package inline

import (
	"context"
	"io/ioutil"
	"os"
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
	type fileStruct struct {
		contents string
		path     string
		mode     os.FileMode
	}
	tests := []struct {
		name            string
		asset           api.InlineAsset
		meta            api.ReleaseMetadata
		templateContext map[string]interface{}
		configGroups    []libyaml.ConfigGroup
		expect          fileStruct
		expectErr       bool
	}{
		{
			name: "happy path",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "foo.txt",
				},
			},
			expect: fileStruct{
				path:     "foo.txt",
				contents: "hello!",
				mode:     0644,
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
		},
		{
			name: "templated dest path",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "{{repl if true}}foo.txt{{repl else}}notfoo.txt{{repl end}}",
					Mode: os.ModePerm,
				},
			},
			expect: fileStruct{
				path:     "foo.txt",
				contents: "hello!",
				mode:     os.ModePerm,
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
		},
		{
			name: "absolute dest path",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "/bin/runc",
				},
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
			expectErr:       true,
		},
		{
			name: "parent dir dest path",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "../../../bin/runc",
				},
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
			expectErr:       true,
		},
		{
			name: "odd filemode",
			asset: api.InlineAsset{
				Contents: "hello!",
				AssetShared: api.AssetShared{
					Dest: "foo.txt",
					Mode: 0543,
				},
			},
			expect: fileStruct{
				path:     "foo.txt",
				contents: "hello!",
				mode:     0543,
			},

			meta:            api.ReleaseMetadata{},
			templateContext: map[string]interface{}{},
			configGroups:    []libyaml.ConfigGroup{},
		},
	}
	for _, test := range tests {
		t.Run(test.name+" aferoFS", func(t *testing.T) {
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
			if !test.expectErr {
				req.NoError(err)
			} else {
				req.Error(err)
			}

			if !test.expectErr {
				contents, err := rootFs.ReadFile(test.expect.path)
				req.NoError(err)
				req.Equal(test.expect.contents, string(contents))
				stat, err := rootFs.Stat(test.expect.path)
				req.NoError(err)
				req.Equal(test.expect.mode, stat.Mode())
			}
		})

		t.Run(test.name+" real FS", func(t *testing.T) {
			req := require.New(t)
			testLogger := &logger.TestLogger{T: t}
			v := viper.New()
			bb := templates.NewBuilderBuilder(testLogger, v, &state.MockManager{})
			tempdir, err := ioutil.TempDir("", "inline-render-test")
			req.NoError(err)
			defer os.RemoveAll(tempdir)
			rootFs := root.Fs{
				Afero:    afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), tempdir)},
				RootPath: "",
			}

			renderer := &LocalRenderer{
				Logger:         testLogger,
				Viper:          v,
				BuilderBuilder: bb,
			}

			err = renderer.Execute(
				rootFs,
				test.asset,
				test.meta,
				test.templateContext,
				test.configGroups,
			)(context.Background())
			if !test.expectErr {
				req.NoError(err)
			} else {
				req.Error(err)
			}

			if !test.expectErr {
				contents, err := rootFs.ReadFile(test.expect.path)
				req.NoError(err)
				req.Equal(test.expect.contents, string(contents))
				stat, err := rootFs.Stat(test.expect.path)
				req.NoError(err)
				req.Equal(test.expect.mode, stat.Mode())
			}
		})
	}
}
