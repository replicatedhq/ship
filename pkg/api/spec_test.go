package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseMetadata_ReleaseName(t *testing.T) {
	tests := []struct {
		name     string
		metadata ReleaseMetadata
		want     string
	}{
		{
			name:     "basic",
			metadata: ReleaseMetadata{},
			want:     "ship",
		},
		{
			name: "channelName",
			metadata: ReleaseMetadata{
				ChannelName: "channel name here",
			},
			want: "channel-name-here",
		},
		{
			name: "metadataName",
			metadata: ReleaseMetadata{
				ShipAppMetadata: ShipAppMetadata{
					Name: "metadata name here",
				},
			},
			want: "metadata-name-here",
		},
		{
			name: "appSlug",
			metadata: ReleaseMetadata{
				AppSlug: "app slug here",
			},
			want: "app-slug-here",
		},
		{
			name: "uppercase",
			metadata: ReleaseMetadata{
				ShipAppMetadata: ShipAppMetadata{
					Name: "UPPERCASE",
				},
			},
			want: "uppercase",
		},
		{
			name: "specials",
			metadata: ReleaseMetadata{
				ShipAppMetadata: ShipAppMetadata{
					Name: "special.characters!aren't+allowed",
				},
			},
			want: "special-characters-aren-t-allowed",
		},
		{
			name: "metadata name overrides channel name",
			metadata: ReleaseMetadata{
				ShipAppMetadata: ShipAppMetadata{
					Name: "metadata",
				},
				ChannelName: "channel",
			},
			want: "metadata",
		},
		{
			name: "channel name overrides app slug",
			metadata: ReleaseMetadata{
				AppSlug:     "appslug",
				ChannelName: "channel",
			},
			want: "channel",
		},
		{
			name: "metadata name overrides app slug",
			metadata: ReleaseMetadata{
				ShipAppMetadata: ShipAppMetadata{
					Name: "metadata",
				},
				AppSlug: "appslug",
			},
			want: "metadata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got := tt.metadata.ReleaseName()
			req.Equal(tt.want, got)
		})
	}
}
