package templates

import (
	"github.com/replicatedcom/ship/pkg/api"
	"text/template"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/spf13/viper"
)

type InstallationContext struct {
	release *api.Release
	viper *viper.Viper
}

func (ctx *InstallationContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"Installation": func(name string) string {
			switch name {
			case "state_file_path":
				return state.Path
			case "customer_id":
				// return ctx.viper.GetString("customer-id")
				return ""
			case "semver":
				return ctx.release.Metadata.Semver
			case "channel_name":
				return ctx.release.Metadata.ChannelName
			case "channel_id":
				return ctx.release.Metadata.ChannelID
			case "release_id":
				return ctx.release.Metadata.ReleaseID
			case "installation_id":
				// return ctx.viper.GetString("installation-id")
				return ""
			case "release_notes":
				return ctx.release.Metadata.ReleaseNotes
			}
			return ""
		},
	}
}