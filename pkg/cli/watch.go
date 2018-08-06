package cli

import (
	"context"

	_ "github.com/kubernetes-sigs/kustomize/pkg/app"
	_ "github.com/kubernetes-sigs/kustomize/pkg/fs"
	_ "github.com/kubernetes-sigs/kustomize/pkg/loader"
	_ "github.com/kubernetes-sigs/kustomize/pkg/resmap"
	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/spf13/cobra"
)

func Watch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch an upstream for updates",
		Long:  `Watch will poll the upstream source for changes, and block until a
change has been published. The watch command will return with an exit code
of 0 when there's an update available.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := ship.Get()
			if err != nil {
				return err
			}

			s.WatchAndExit(context.Background())
			return nil
		},
	}

	return cmd
}
