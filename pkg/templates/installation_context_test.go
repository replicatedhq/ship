package templates

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/testing/logger"
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
	tests := testCases()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assertions := require.New(t)

			v := viper.New()
			ctx := &InstallationContext{
				Meta:   test.Meta,
				Viper:  v,
				Logger: &logger.TestLogger{T: t},
			}
			v.Set("customer-id", "abc")
			v.Set("installation-id", "xyz")

			builderBuilder := &BuilderBuilder{
				Viper:  v,
				Logger: &logger.TestLogger{T: t},
			}

			builder := builderBuilder.NewBuilder(ctx)

			built, err := builder.String(test.Tpl)
			assertions.NoError(err, "executing template")

			assertions.Equal(test.Expected, built)
		})
	}
}

func testCases() []TestInstallation {
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
			Name:     "state_file_path",
			Meta:     api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "state_file_path" }}`,
			Expected: "It's " + constants.StatePath,
		},
		{
			Name:     "customer_id",
			Meta:     api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "customer_id" }}`,
			Expected: `It's abc`,
		},
		{
			Name:     "installation_id",
			Meta:     api.ReleaseMetadata{},
			Tpl:      `It's {{repl Installation "installation_id" }}`,
			Expected: `It's xyz`,
		},
		{
			Name: "entitlement value",
			Meta: api.ReleaseMetadata{
				Entitlements: api.Entitlements{
					Values: []api.EntitlementValue{
						{
							Key:   "num_seats",
							Value: "3",
						},
					},
				}},
			Tpl:      `You get {{repl EntitlementValue "num_seats" }} seats`,
			Expected: `You get 3 seats`,
		},
		{
			Name:     "no entitlements",
			Meta:     api.ReleaseMetadata{},
			Tpl:      `You get {{repl EntitlementValue "num_repos" }} repos`,
			Expected: `You get  repos`,
		},
		{
			Name: "entitlement value not found",
			Meta: api.ReleaseMetadata{
				Entitlements: api.Entitlements{
					Values: []api.EntitlementValue{
						{
							Key:   "num_seats",
							Value: "3",
						},
					},
				}},
			Tpl:      `You get {{repl EntitlementValue "num_repos" }} repos`,
			Expected: `You get  repos`,
		},
	}
	return tests
}
