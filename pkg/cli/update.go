package cli

import (
	"context"
	"strings"

	_ "github.com/kubernetes-sigs/kustomize/pkg/app"
	_ "github.com/kubernetes-sigs/kustomize/pkg/fs"
	_ "github.com/kubernetes-sigs/kustomize/pkg/loader"
	_ "github.com/kubernetes-sigs/kustomize/pkg/resmap"
	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Update() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updated a chart",
		Long:  `Updated a chart`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := ship.Get()
			if err != nil {
				return err
			}

			s.UpdateAndMaybeExit(context.Background())
			return nil
		},
		Hidden: true,
	}

	cmd.Flags().String("headed", "", "Run ship update in a graphical user interface.")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// default to headless if user doesn't set --headed=true
	if !viper.GetBool("headed") {
		viper.Set("headless", true)
	}

	return cmd
}
