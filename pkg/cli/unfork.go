package cli

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Unfork() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfork FORK",
		Short: "Unfork a previously forked Helm chart or Kubernetes YAML ",
		Long:  `Given an forked and modified Helm chart or Kubernetes YAML, create patches for the changes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				_ = cmd.Help()
				return errors.New("Error: please supply a fork")
			}

			v.Set("fork", args[0])
			v.Set("headless", true) // We don't support headed unforking

			s, err := ship.Get(v)
			if err != nil {
				return err
			}

			return s.UnforkAndMaybeExit(context.Background())
		},
	}

	cmd.Flags().StringP("upstream", "", "", "path to the upstream")

	_ = viper.BindPFlags(cmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
