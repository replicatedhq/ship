package templates

import (
	"testing"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type TestInstallation struct {
	Name     string
	Meta     api.ReleaseMetadata
	Tpl      string
	Expected string
	Viper    *viper.Viper
}

func TestInstallationContext(t *testing.T) {
	tests := []TestInstallation{
		{
			Name: "semver",
			Meta: api.ReleaseMetadata{
				Semver: "1.0.0",
			},
			Tpl:      `It's {{repl Installation "semver" }}`,
			Expected: `It's 1.0.0`,
		},
		{
			Name: "channel_name",
			Meta: api.ReleaseMetadata{
				ChannelName: "brisket",
			},
			Tpl:      `It's {{repl Installation "channel_name" }}`,
			Expected: `It's brisket`,
		},
		{
			Name: "channel_id",
			Meta: api.ReleaseMetadata{
				ChannelID: "123",
			},
			Tpl:      `It's {{repl Installation "channel_id" }}`,
			Expected: `It's 123`,
		},
		{
			Name: "release_id",
			Meta: api.ReleaseMetadata{
				ReleaseID: "123",
			},
			Tpl:      `It's {{repl Installation "release_id" }}`,
			Expected: `It's 123`,
		},
		{
			Name: "release_notes",
			Meta: api.ReleaseMetadata{
				ReleaseNotes: "corn bread",
			},
			Tpl:      `It's {{repl Installation "release_notes" }}`,
			Expected: `It's corn bread`,
		},
		{
			Name: "state_file_path",
			Meta: api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "state_file_path" }}`,
			Expected: `It's .ship/state.json`,
		},
		{
			Name: "customer_id",
			Meta: api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "customer_id" }}`,
			Expected: `It's abc`,
		},
		{
			Name: "installation_id",
			Meta: api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "installation_id" }}`,
			Expected: `It's xyz`,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assertions := require.New(t)

			ctx := &InstallationContext{
				Meta:  test.Meta,
				Viper: viper.New(),
			}
			ctx.Viper.Set("customer-id", "abc")
			ctx.Viper.Set("installation-id", "xyz")

			builder := NewBuilder(ctx)

			built, err := builder.String(test.Tpl)
			assertions.NoError(err, "executing template")

			assertions.Equal(test.Expected, built)
		})
	}
}
