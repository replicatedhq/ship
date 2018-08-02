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

	// todo figure out why we're not getting this from root cmd
	cmd.PersistentFlags().String("state-file", "", "path to the state file to read from, defaults to .ship/state.json")

	cmd.PersistentFlags().StringP("customer-endpoint", "e", "https://pg.replicated.com/graphql", "Upstream application spec server address")

	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
