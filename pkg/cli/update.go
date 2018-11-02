package cli

import (
	"context"
	"strings"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Update() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Pull an updated helm chart",
		Long:  `Pull an updated helm chart to be integrated into current application configuration`,
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

	viper.BindPFlags(cmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
