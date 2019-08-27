package cli

import (
	"context"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Update() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Fetch the latest upstream for a Ship application",
		Long:  `Given an existing Ship state.json, fetch the latest upstream and merge`,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
			_ = viper.BindPFlags(cmd.PersistentFlags())
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			// update should run in headless mode by default
			viper.Set("isUpdate", true)

			if !viper.GetBool("headed") {
				viper.Set("headless", true)
			}

			s, err := ship.Get(viper.GetViper())
			if err != nil {
				return err
			}

			return s.UpdateAndMaybeExit(context.Background())
		},
	}

	cmd.Flags().BoolP("headed", "", false, "run ship update in headed mode")

	return cmd
}
