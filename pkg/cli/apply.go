package cli

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/ship"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ApplyCmd runs the core Ship workflow
func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "manage and generate on-prem applications for use with Ship",
		Long: `ship apply will generate assets
for installing an installation
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := ship.FromViper(viper.GetViper())
			if err != nil {
				return errors.Wrap(err, "initialize")
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
