package cli

import (
	"context"
	"strings"

	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init UPSTREAM",
		Short: "Build and deploy kubernetes applications with kustomize.",
		Long: `
Build and deploy applications to be integrated
with a gitops style workflow.


Upstream can be one of:

- A path to Kubernetes manifests in a github repo [github.com/replicatedhq/test-charts/plain-k8s]
- A path to a helm chart in a github repo         [github.com/helm/charts/stable/anchore-engine]
- A path to a specific "ref" to a helm chart or
  Kubernetes manifests in a github repo           [github.com/helm/charts/tree/abcdef123456/stable/anchore-engine]
- A go-getter compatible URL
  (github.com/hashicorp/go-getter)              [git::gitlab.com/myrepo/mychart, ./local-charts/nginx-ingress, github.com/myrepo/mychart?ref=abcdef123456//my/path]
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			if len(args) != 0 {
				v.Set("target", args[0])
			}
			s, err := ship.Get(v)
			if err != nil {
				return err
			}

			return s.InitAndMaybeExit(context.Background())
		},
	}

	cmd.Flags().String("file", "", "File path to helm chart")

	cmd.Flags().Bool("rm-asset-dest", false, "Always remove asset destinations if already present")
	cmd.Flags().Int("retries", 3, "Number of times to retry retrieving upstream")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
