package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/api"
)

func TestState_ReleaseMetadata(t *testing.T) {
	tests := []struct {
		name string
		V1   *V1
		want *api.ReleaseMetadata
	}{
		{
			name: "all nil",
			V1:   nil,
			want: nil,
		},
		{
			name: "no upstream contents",
			V1:   &V1{ReleaseName: "no upstream contents"},
			want: nil,
		},
		{
			name: "basic upstream contents",
			V1: &V1{
				UpstreamContents: &UpstreamContents{
					AppRelease: &ShipRelease{ID: "abc"},
				},
			},
			want: &api.ReleaseMetadata{
				ReleaseID:      "abc",
				Images:         []api.Image{},
				GithubContents: []api.GithubContent{},
			},
		},
		{
			name: "upstream contents with metadata",
			V1: &V1{
				UpstreamContents: &UpstreamContents{
					AppRelease: &ShipRelease{ID: "abc"},
				},
				Metadata: &Metadata{
					CustomerID: "xyz",
					License: License{
						ID: "licenseID",
					},
				},
			},
			want: &api.ReleaseMetadata{
				ReleaseID:      "abc",
				Images:         []api.Image{},
				GithubContents: []api.GithubContent{},
				CustomerID:     "xyz",
				License: api.License{
					ID: "licenseID",
				},
			},
		},
		{
			name: "upstream contents with actual contents",
			V1: &V1{
				UpstreamContents: &UpstreamContents{
					AppRelease: &ShipRelease{
						ID: "abc",
						GithubContents: []GithubContent{
							{
								Repo: "testRepo",
								Path: "testPath",
								Ref:  "testRef",
								Files: []GithubFile{
									{
										Name: "testFileName",
										Path: "testFilePath",
										Sha:  "testFileSha",
										Size: 1234,
										Data: "testFileData",
									},
								},
							},
						},
					},
				},
			},
			want: &api.ReleaseMetadata{
				ReleaseID: "abc",
				Images:    []api.Image{},
				GithubContents: []api.GithubContent{
					{
						Repo: "testRepo",
						Path: "testPath",
						Ref:  "testRef",
						Files: []api.GithubFile{
							{
								Name: "testFileName",
								Path: "testFilePath",
								Sha:  "testFileSha",
								Size: 1234,
								Data: "testFileData",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			v := State{
				V1: tt.V1,
			}
			got := v.ReleaseMetadata()

			req.Equal(tt.want, got)
		})
	}
}
