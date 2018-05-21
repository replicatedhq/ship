package templates

import (
	"github.com/replicatedcom/ship/pkg/api"
	"text/template"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
)

func NewInstallContext() *InstallationContext {
	// TODO
	return nil
}

type InstallationContext struct {
	Release *api.Release
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
				return ctx.Release.Metadata.Semver
			case "channel_name":
				return ctx.Release.Metadata.ChannelName
			case "channel_id":
				return ctx.Release.Metadata.ChannelID
			case "release_id":
				return ctx.Release.Metadata.ReleaseID
			case "installation_id":
				// return ctx.viper.GetString("installation-id")
				return ""
			case "release_notes":
				return ctx.Release.Metadata.ReleaseNotes
			}
			return ""
		},
	}
}