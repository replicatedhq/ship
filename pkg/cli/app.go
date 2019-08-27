package cli

import (
	"context"
	"fmt"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func App() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "app",
		Short:  "Download and configure a licensed third party application",
		Long:   `Download and configure a third party application using a supplied customer id.`,
		Hidden: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
			_ = viper.BindPFlags(cmd.PersistentFlags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set("is-app", true)
			s, err := ship.Get(viper.GetViper())
			if err != nil {
				return err
			}
			return s.ExecuteAndMaybeExit(context.Background())
		},
	}

	// required
	cmd.Flags().String("customer-id", "", "Customer ID for which to query app specs. Required for all ship operations.")
	cmd.Flags().StringP("installation-id", "i", "", "Installation ID for which to query app specs")

	// optional
	cmd.Flags().String("channel-id", "", "ship channel to install from")
	cmd.Flags().String("release-id", "", "specific Release ID to pin installation to.")
	cmd.Flags().String("release-semver", "", "specific release version to pin installation to. Requires channel-id")
	cmd.Flags().Bool("terraform-yes", false, "Automatically answer \"yes\" to all terraform prompts")

	// optional developer flags
	cmd.Flags().String("runbook", "", developerFlagUsage)
	cmd.Flags().String("set-channel-name", "", developerFlagUsage)
	cmd.Flags().String("set-channel-icon", "", developerFlagUsage)
	cmd.Flags().StringSlice("set-github-contents", []string{}, fmt.Sprintf("Specify a REPO:REPO_PATH:REF:LOCAL_PATH to override github checkouts to use a local path on the filesystem. %s. ", developerFlagUsage))
	cmd.Flags().String("set-entitlements-json", "{\"values\":[]}", fmt.Sprintf("Specify json for entitlements payload. %s", developerFlagUsage))

	// Deprecated developer flags
	cmd.Flags().String("studio-file", "", developerFlagUsage)
	_ = cmd.Flags().MarkDeprecated("studio-file", "please upgrade to the --runbook flag")
	cmd.Flags().String("studio-channel-name", "", developerFlagUsage)
	_ = cmd.Flags().MarkDeprecated("studio-channel-name", "please upgrade to the --set-channel-name flag")
	cmd.Flags().String("studio-channel-icon", "", developerFlagUsage)
	_ = cmd.Flags().MarkDeprecated("studio-channel-icon", "please upgrade to the --set-channel-icon flag")

	return cmd
}
