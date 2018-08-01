package cli

import (
	"context"
	"strings"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	developerFlagUsage = "Useful for debugging your specs on the command line, without having to make round trips to the server"
)

func App() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Download and configure a licensed third party application",
		Long:  `Download and configure a third party application using a supplied customer id.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ship.RunE(context.Background())
		},
	}

	// required
	cmd.Flags().String("customer-id", "", "Customer ID for which to query app specs. Required for all ship operations.")
	cmd.Flags().StringP("installation-id", "i", "", "Installation ID for which to query app specs")

	// optional
	cmd.Flags().String("channel-id", "", "ship channel to install from")
	cmd.Flags().StringP("customer-endpoint", "e", "https://pg.replicated.com/graphql", "Upstream application spec server address")
	cmd.Flags().String("release-id", "", "specific Release ID to pin installation to.")
	cmd.Flags().String("release-semver", "", "specific release version to pin installation to. Requires channel-id")
	cmd.Flags().Bool("terraform-yes", false, "Automatically answer \"yes\" to all terraform prompts")

	// optional developer flags
	cmd.Flags().String("runbook", "", developerFlagUsage)
	cmd.Flags().String("set-channel-name", "", developerFlagUsage)
	cmd.Flags().String("set-channel-icon", "", developerFlagUsage)

	// Deprecated developer flags
	cmd.Flags().String("studio-file", "", developerFlagUsage)
	cmd.Flags().MarkDeprecated("studio-file", "please upgrade to the --runbook flag")
	cmd.Flags().String("studio-channel-name", "", developerFlagUsage)
	cmd.Flags().MarkDeprecated("studio-channel-name", "please upgrade to the --set-channel-name flag")
	cmd.Flags().String("studio-channel-icon", "", developerFlagUsage)
	cmd.Flags().MarkDeprecated("studio-channel-icon", "please upgrade to the --set-channel-icon flag")

	viper.BindPFlags(cmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
