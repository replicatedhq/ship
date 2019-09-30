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
			Name: "customer_id",
			Meta: api.ReleaseMetadata{
				CustomerID: "abc",
			},
			Tpl:      `It's {{repl Installation "customer_id" }}`,
			Expected: `It's abc`,
		},
		{
			Name: "installation_id",
			Meta: api.ReleaseMetadata{
				InstallationID: "xyz",
			},
			Tpl:      `It's {{repl Installation "installation_id" }}`,
			Expected: `It's xyz`,
		},
		{
			Name: "installation_id == license_id",
			Meta: api.ReleaseMetadata{
				InstallationID: "xyz",
			},
			Tpl:      `It's {{repl Installation "license_id" }}`,
			Expected: `It's xyz`,
		},
		{
			Name: "license_id",
			Meta: api.ReleaseMetadata{
				LicenseID: "myLicenseID",
			},
			Tpl:      `It's {{repl Installation "license_id" }}`,
			Expected: `It's myLicenseID`,
		},
		{
			Name: "license_id == installation_id",
			Meta: api.ReleaseMetadata{
				LicenseID: "myLicenseID",
			},
			Tpl:      `It's {{repl Installation "installation_id" }}`,
			Expected: `It's myLicenseID`,
		},
		{
			Name: "app_slug",
			Meta: api.ReleaseMetadata{
				AppSlug: "my_app_slug",
			},
			Tpl:      `It's {{repl Installation "app_slug" }}`,
			Expected: `It's my_app_slug`,
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
			Name: "entitlement value == license field value",
			Meta: api.ReleaseMetadata{
				Entitlements: api.Entitlements{
					Values: []api.EntitlementValue{
						{
							Key:   "num_seats",
							Value: "3",
						},
					},
				}},
			Tpl:      `You get {{repl LicenseFieldValue "num_seats" }} seats`,
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
		{
			Name: "collect spec",
			Meta: api.ReleaseMetadata{
				CollectSpec: `collect:
  v1:
    - {}`,
			},
			Tpl: `{{repl CollectSpec }}`,
			Expected: `collect:
  v1:
    - {}`,
		},
		{
			Name: "analyze spec",
			Meta: api.ReleaseMetadata{
				AnalyzeSpec: `analyze:
  v1:
    - {}`,
			},
			Tpl: `{{repl AnalyzeSpec }}`,
			Expected: `analyze:
  v1:
    - {}`,
		},
		{
			Name: "ship customer release",
			Meta: api.ReleaseMetadata{
				AnalyzeSpec: `this is not an analyze spec`,
				ConfigSpec:  ``,
				CollectSpec: ``,
				GithubContents: []api.GithubContent{{
					Repo: "abc",
					Path: "xyz",
					Ref:  "test",
					Files: []api.GithubFile{{
						Name: "abc",
						Path: "xyz",
						Sha:  "123",
						Size: 456,
						Data: "789",
					}},
				}},
				Images: []api.Image{{
					URL:      "abc",
					Source:   "xyz",
					AppSlug:  "123",
					ImageKey: "456",
				}},
				LicenseID: "myLicenseID",
			},
			Tpl: `{{repl ShipCustomerRelease }}`,
			Expected: `releaseId: ""
sequence: 0
customerId: ""
installation: ""
channelId: ""
appSlug: ""
licenseId: myLicenseID
channelName: ""
channelIcon: ""
semver: ""
releaseNotes: ""
created: ""
installed: ""
registrySecret: ""
images: []
githubContents: []
shipAppMetadata:
  description: ""
  version: ""
  icon: ""
  name: ""
  readme: ""
  url: ""
  contentSHA: ""
  releaseNotes: ""
entitlements:
  meta:
    lastupdated: 0001-01-01T00:00:00Z
    customerid: ""
  serialized: ""
  signature: ""
  values: []
  utilizations: []
entitlementSpec: ""
configSpec: ""
collectSpec: ""
analyzeSpec: ""
type: ""
license:
  id: ""
  assignee: ""
  createdAt: 0001-01-01T00:00:00Z
  expiresAt: 0001-01-01T00:00:00Z
  type: ""
`,
		},
		{
			Name: "ship customer release full",
			Meta: api.ReleaseMetadata{
				AnalyzeSpec: `this is not an analyze spec`,
				ConfigSpec:  `this is not a config spec`,
				CollectSpec: `this is not a collect spec`,
				GithubContents: []api.GithubContent{{
					Repo: "abc",
					Path: "xyz",
					Ref:  "test",
					Files: []api.GithubFile{{
						Name: "abc",
						Path: "xyz",
						Sha:  "123",
						Size: 456,
						Data: "789",
					}},
				}},
				Images: []api.Image{{
					URL:      "abc",
					Source:   "xyz",
					AppSlug:  "123",
					ImageKey: "456",
				}},
				LicenseID: "myLicenseID",
			},
			Tpl: `{{repl ShipCustomerReleaseFull }}`,
			Expected: `releaseId: ""
sequence: 0
customerId: ""
installation: ""
channelId: ""
appSlug: ""
licenseId: myLicenseID
channelName: ""
channelIcon: ""
semver: ""
releaseNotes: ""
created: ""
installed: ""
registrySecret: ""
images:
- url: abc
  source: xyz
  appSlug: "123"
  imageKey: "456"
githubContents:
- repo: abc
  path: xyz
  ref: test
  files:
  - name: abc
    path: xyz
    sha: "123"
    size: 456
    data: "789"
shipAppMetadata:
  description: ""
  version: ""
  icon: ""
  name: ""
  readme: ""
  url: ""
  contentSHA: ""
  releaseNotes: ""
entitlements:
  meta:
    lastupdated: 0001-01-01T00:00:00Z
    customerid: ""
  serialized: ""
  signature: ""
  values: []
  utilizations: []
entitlementSpec: ""
configSpec: this is not a config spec
collectSpec: this is not a collect spec
analyzeSpec: this is not an analyze spec
type: ""
license:
  id: ""
  assignee: ""
  createdAt: 0001-01-01T00:00:00Z
  expiresAt: 0001-01-01T00:00:00Z
  type: ""
`,
		},
	}
	return tests
}
