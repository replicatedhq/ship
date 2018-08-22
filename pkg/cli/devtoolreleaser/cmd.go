package devtoolreleaser

import (
	"strings"

	"context"

	"os"

	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/fs"
	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Cmd() *cobra.Command {
	vip := viper.New()
	cmd := &cobra.Command{
		Use:   "devtool-releaser",
		Short: "API client for creating ship releases",
		Long: `

`,
		RunE: func(cmd *cobra.Command, args []string) error {

			releaser := &Releaser{
				viper:  vip,
				logger: logger.New(vip, fs.NewBaseFilesystem()),
				ui: &cli.ColoredUi{
					OutputColor: cli.UiColorNone,
					ErrorColor:  cli.UiColorRed,
					WarnColor:   cli.UiColorYellow,
					InfoColor:   cli.UiColorGreen,
					Ui: &cli.BasicUi{
						Reader:      os.Stdin,
						Writer:      os.Stdout,
						ErrorWriter: os.Stderr,
					},
				},
			}

			err := releaser.Release(context.Background())

			if err != nil {
				return errors.Wrap(err, "promote release")
			}

			return nil
		},
	}

	cmd.Flags().String("vendor-token", "", "Token to use to communicate with https://g.replicated.com")
	cmd.Flags().String("graphql-api-address", "https://g.replicated.com/graphql", "upstream g. address")
	cmd.Flags().String("spec-file", "", "spec file to promote")
	cmd.Flags().String("channel-id", "", "channel id to promote")
	cmd.Flags().String("semver", "", "semver of the release")
	cmd.Flags().String("release-notes", "", "release notes")
	cmd.Flags().String("log-level", "off", "log level")

	vip.BindPFlags(cmd.Flags())
	vip.BindPFlags(cmd.PersistentFlags())
	vip.AutomaticEnv()
	vip.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
