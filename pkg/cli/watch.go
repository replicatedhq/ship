package cli

import (
	"context"
	"time"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Watch() *cobra.Command {
	v := viper.GetViper()
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch an upstream for updates",
		Long: `Watch will poll the upstream source for changes, and block until a
change has been published. The watch command will return with an exit code
of 0 when there's an update available.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
			_ = viper.BindPFlags(cmd.PersistentFlags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := ship.Get(v)
			if err != nil {
				return err
			}

			return s.WatchAndExit(context.Background())
		},
	}

	cmd.Flags().DurationP("interval", "", time.Duration(time.Minute*15), "interval to wait between cycles polling for updates")
	cmd.Flags().BoolP("exit", "", false, "exit immediately after first poll, regardless of whether an update is available")

	return cmd
}
