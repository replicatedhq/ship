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

func Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [CHART]",
		Short: "Build and deploy kustomize configured helm charts",
		Long: `Build and deploy kustomize configured helm charts to be integrated
with a gitops style workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			if len(args) != 0 {
				v.Set("target", args[0])
			}
			s, err := ship.Get(v)
			if err != nil {
				return err
			}

			s.InitAndMaybeExit(context.Background())
			return nil
		},
	}

	cmd.Flags().String("file", "", "File path to helm chart")

	cmd.Flags().String("raw", "", "File path to already rendered kubernetes YAML. Intended for use with non-helm K8s YAML or with a helm chart that has already been templated.")
	cmd.Flags().String("chartRepoURL", "", "Helm chart repo URL as would be passed to `helm fetch` with the --repo flag")
	cmd.Flags().String("chartVersion", "", "Helm chart version as would be passed to `helm fetch` with the --version flag")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
