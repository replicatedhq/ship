package planner

import (
	"testing"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
)

type TestWebAsset struct {
	Name    string
	Release *api.Release
}

func TestWebAssetStep(t *testing.T) {
	tests := []TestWebAsset{
		{
			Name: "test",
			Release: &api.Release{
				Spec: api.Spec{
					Assets: api.Assets{
						V1: []api.Asset{{
							Web: &api.WebAsset{
								URL: "https://www.google.com",
								AssetShared: api.AssetShared{
									Dest: "./installer/google.html",
								},
							},
						}},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			fakeFS := afero.Afero{Fs: afero.NewMemMapFs()}

			contents, _ := fakeFS.ReadFile(test.Release.Spec.Assets.V1[0].Web.AssetShared.Dest)
		})
	}
}
