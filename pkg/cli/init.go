package cli

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	developerFlagUsage = "Useful for debugging your specs on the command line, without having to make round trips to the server"
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
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
			_ = viper.BindPFlags(cmd.PersistentFlags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			if len(args) == 0 {
				_ = cmd.Help()
				return errors.New("Error: please supply an upstream")
			}

			v.Set("upstream", args[0])
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
	cmd.Flags().Bool("preserve-state", false, "Skips prompt to remove existing state. If an existing state file is present, ship update --headed lifecycle will be used.")

	// optional developer flags for "ship init replicated.app"
	cmd.Flags().String("set-channel-name", "", developerFlagUsage)
	cmd.Flags().String("set-channel-icon", "", developerFlagUsage)
	cmd.Flags().StringSlice("set-github-contents", []string{}, fmt.Sprintf("Specify a REPO:REPO_PATH:REF:LOCAL_PATH to override github checkouts to use a local path on the filesystem. %s. ", developerFlagUsage))
	cmd.Flags().String("set-entitlements-json", "{\"values\":[]}", fmt.Sprintf("Specify json for entitlements payload. %s", developerFlagUsage))

	return cmd
}
