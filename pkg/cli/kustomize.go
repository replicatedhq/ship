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
			if len(args) != 0 {
				viper.Set("chart", args[0])
			}
			s, err := ship.Get()
			if err != nil {
				return err
			}
			s.IsKustomize = true
			s.ExecuteAndMaybeExit(context.Background())
			return nil
		},
	}

	cmd.Flags().String("file", "", "File path to helm chart")

	cmd.Flags().String("raw", "", "File path to already rendered kubernetes YAML. Intended for use with non-helm K8s YAML or with a helm chart that has already been templated.")
	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
