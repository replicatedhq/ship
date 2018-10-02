package cli

import (
	"context"
	"time"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "sigs.k8s.io/kustomize/pkg/app"
	_ "sigs.k8s.io/kustomize/pkg/fs"
	_ "sigs.k8s.io/kustomize/pkg/loader"
	_ "sigs.k8s.io/kustomize/pkg/resmap"
)

func Watch() *cobra.Command {
	v := viper.GetViper()
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch an upstream for updates",
		Long: `Watch will poll the upstream source for changes, and block until a
change has been published. The watch command will return with an exit code
of 0 when there's an update available.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := ship.Get(v)
			if err != nil {
				return err
			}

			return s.WatchAndExit(context.Background())
		},
	}

	cmd.Flags().DurationP("interval", "", time.Duration(time.Minute*15), "interval to wait between cycles polling for updates")

	v.BindPFlags(cmd.PersistentFlags())
	v.BindPFlags(cmd.Flags())

	return cmd
}
