package cli

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// PlanCmd runs the core Ship workflow, but sets a flag to disable asset generation and
// state management
func PlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "manage and serve on-prem ship data",
		Long: `ship plan can be used to execute a dry run
of an application installation, without generating any assets or modifying
state.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := ship.FromViper(viper.GetViper())
			rc.PlanOnly = true
			if err != nil {
				return errors.Wrap(err, "initialize daemon")
			}
			err = rc.Execute(context.Background())
			if err != nil {
				rc.OnError(err)
			}
			return err
		},
	}

	return cmd
}
