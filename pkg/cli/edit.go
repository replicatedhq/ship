package cli

import (
	"context"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Edit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit settings for the current version of a Ship application",
		Long:  `Given an existing Ship state.json, use the stored upstream state to edit settings without updating the version`,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
			_ = viper.BindPFlags(cmd.PersistentFlags())
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set("isEdit", true)

			s, err := ship.Get(viper.GetViper())
			if err != nil {
				return err
			}

			return s.UpdateAndMaybeExit(context.Background())
		},
	}

	return cmd
}
