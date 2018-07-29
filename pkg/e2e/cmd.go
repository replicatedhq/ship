package e2e

import (
	"testing"

	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "e2e",
		Short:  "e2e testing for ship",
		Hidden: true,
		Long: `

`,
		Run: func(cmd *cobra.Command, args []string) {

			runner := &Runner{}
			testing.Main(func(pat, str string) (bool, error) { return true, nil },
				[]testing.InternalTest{
					{Name: "E2E", F: func(t *testing.T) { runner.Run(t) }},
				},
				[]testing.InternalBenchmark{},
				[]testing.InternalExample{},
			)
		},
	}

	cmd.Flags().String("vendor-token", "", "Token to use to communicate with http://g.replicated.com")
	cmd.Flags().String("graphql-api-address", "", "upstream g. address")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
