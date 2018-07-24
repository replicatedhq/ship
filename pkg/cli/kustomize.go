package cli

import (
	"context"
	"strings"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Kustomize() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kustomize",
		Short: "Build and deploy kustomize configured helm charts",
		Long: `Build and deploy kustomize configured helm charts to be integrated
with a git ops style workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ship.RunE(context.Background())
		},
	}

	cmd.Flags().String("chart", "", "Link to github with helm chart")
	cmd.MarkFlagRequired("chart")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
