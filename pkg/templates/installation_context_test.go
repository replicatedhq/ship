package templates

import (
	"testing"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/stretchr/testify/require"
)

type TestInstallation struct {
	Name string
	Release *api.Release
	Tpl string
	Expected string
}

func TestInstallationContext(t *testing.T) {
	tests := []TestInstallation{
		{
			Name: "semver",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{
					Semver: "1.0.0",
				},
			},
			Tpl:      `It's {{repl Installation "semver" }}`,
			Expected: `It's 1.0.0`,
		},
		{
			Name: "channel_name",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{
					ChannelName: "brisket",
				},
			},
			Tpl:      `It's {{repl Installation "channel_name" }}`,
			Expected: `It's brisket`,
		},
		{
			Name: "channel_id",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{
					ChannelID: "123",
				},
			},
			Tpl:      `It's {{repl Installation "channel_id" }}`,
			Expected: `It's 123`,
		},
		{
			Name: "release_id",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{
					ReleaseID: "123",
				},
			},
			Tpl:      `It's {{repl Installation "release_id" }}`,
			Expected: `It's 123`,
		},
		{
			Name: "release_notes",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{
					ReleaseNotes: "corn bread",
				},
			},
			Tpl:      `It's {{repl Installation "release_notes" }}`,
			Expected: `It's corn bread`,
		},
		{
			Name: "state_file_path",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{},
			},
			Tpl:      `It's {{repl Installation "state_file_path" }}`,
			Expected: `It's .ship/state.json`,
		},
		{
			Name: "customer_id",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{},
			},
			Tpl:      `It's {{repl Installation "customer_id" }}`,
			Expected: `It's `,
		},
		{
			Name: "installation_id",
			Release: &api.Release{
				Metadata: api.ReleaseMetadata{},
			},
			Tpl:      `It's {{repl Installation "installation_id" }}`,
			Expected: `It's `,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assertions := require.New(t)

			ctx := &InstallationContext{
			    release: test.Release,
			}

			builder := NewBuilder(ctx)

			built, err := builder.String(test.Tpl)
			assertions.NoError(err, "executing template")

			assertions.Equal(test.Expected, built)
		})
	}
}
