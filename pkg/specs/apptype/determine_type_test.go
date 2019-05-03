package apptype

import (
	"context"
	"testing"

	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func Test_inspector_fetchEditFiles(t *testing.T) {
	tests := []struct {
		name    string
		state   state.UpstreamContents
		wantApp LocalAppCopy
		wantErr bool
	}{
		{
			name: "app release contents",
			state: state.UpstreamContents{
				AppRelease: &state.ShipRelease{},
			},
			wantApp: &localAppCopy{AppType: "replicated.app"},
			wantErr: false,
		},
		{
			name: "k8s yaml contents",
			state: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "test.yaml", FileContents: "YTogYg=="},
				},
			},
			wantApp: &localAppCopy{AppType: "k8s"},
			wantErr: false,
		},
		{
			name: "helm chart yaml contents",
			state: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "test.yaml", FileContents: "YTogYg=="},
					{FilePath: "Chart.yaml", FileContents: "VGhpcyBpcyBwcm9iYWJseSBub3QgYSB2YWxpZCBoZWxtIGNoYXJ0IHlhbWwgZmlsZQ=="},
				},
			},
			wantApp: &localAppCopy{AppType: "helm"},
			wantErr: false,
		},
		{
			name: "inline replicated app contents",
			state: state.UpstreamContents{
				UpstreamFiles: []state.UpstreamFile{
					{FilePath: "test.yaml", FileContents: "YTogYg=="},
					{FilePath: "ship.yaml", FileContents: "SSBBTSBTSElQ"},
				},
			},
			wantApp: &localAppCopy{AppType: "inline.replicated.app"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			tlog := logger.TestLogger{T: t}

			vip := viper.New()
			vip.Set("isEdit", true)

			manager := state.NewManager(&tlog, fs, vip)
			err := manager.SerializeUpstreamContents(&tt.state)
			req.NoError(err)

			i := NewInspector(&tlog, fs, vip, manager, nil).(*inspector)

			gotApp, err := i.fetchEditFiles(context.Background())
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}

			req.Equal(tt.wantApp.GetType(), gotApp.GetType())
		})
	}
}
