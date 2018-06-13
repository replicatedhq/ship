package templates

import (
	"text/template"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/spf13/viper"
)

type InstallationContext struct {
	Meta  api.ReleaseMetadata
	Viper *viper.Viper
}

func (ctx *InstallationContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"Installation": func(name string) string {
			switch name {
			case "state_file_path":
				return state.Path
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
