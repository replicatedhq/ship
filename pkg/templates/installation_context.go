package templates

import (
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/viper"
)

type InstallationContext struct {
	Meta   api.ReleaseMetadata
	Viper  *viper.Viper
	Logger log.Logger
}

func (ctx *InstallationContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"EntitlementValue": func(name string) string {
			if ctx.Meta.Entitlements.Values == nil {
				level.Debug(ctx.Logger).Log("event", "EntitlementValue.empty")
				return ""
			}

			for _, value := range ctx.Meta.Entitlements.Values {
				if value.Key == name {
					return value.Value
				}
			}

			level.Debug(ctx.Logger).Log("event", "EntitlementValue.notFound", "key", name, "values.count", len(ctx.Meta.Entitlements.Values))
			return ""
		},

		"Installation": func(name string) string {
			switch name {
			case "state_file_path":
				return constants.StatePath
			case "customer_id":
				return ctx.Viper.GetString("customer-id")
			case "semver":
				return ctx.Meta.Semver
			case "channel_name":
				return ctx.Meta.ChannelName
			case "channel_id":
				return ctx.Meta.ChannelID
			case "release_id":
				return ctx.Meta.ReleaseID
			case "installation_id":
				return ctx.Viper.GetString("installation-id")
			case "release_notes":
				return ctx.Meta.ReleaseNotes
			}
			return ""
		},
	}
}
