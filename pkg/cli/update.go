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

	return cmd
}
