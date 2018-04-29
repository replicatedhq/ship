package cli

import (
	"fmt"
	"os"

	"context"

	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/cli/devtool_releaser"
	"github.com/replicatedcom/ship/pkg/e2e"
	"github.com/replicatedcom/ship/pkg/ship"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/replicatedcom/ship/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ship",
		Short: "manage and serve on-prem ship data",
		Long: `ship allows for managing and securely delivering
application specs to be used in on-prem installations.
`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			version.Init()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rc, err := ship.FromViper(viper.GetViper())
			if err != nil {
				return errors.Wrap(err, "initialize")
			}
			err = rc.Execute(context.Background())
			if err != nil {
				rc.ExitWithError(err)
			}
			return nil // let ExitWithError handle it ^
		},
	}
	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/replicated/ship.yaml)")
	cmd.PersistentFlags().String("log-level", "off", "Log level")
	cmd.PersistentFlags().StringP("customer-endpoint", "e", "https://pg.replicated.com/graphql", "Upstream application spec server address")

	// required
	cmd.PersistentFlags().String("customer-id", "", "Customer ID for which to query app specs. Required for all ship operations.")

	// optional
	cmd.PersistentFlags().String("release-id", "", "specific Release ID to pin installation to.")
	cmd.PersistentFlags().String("release-semver", "", "specific release version to pin installation to. Requires channel-id")
	cmd.PersistentFlags().String("channel-id", "", "ship channel to install from")
	cmd.PersistentFlags().StringP("installation-id", "i", "", "Installation ID for which to query app specs")

	if specs.AllowInlineSpecs {
		cmd.PersistentFlags().StringP("studio-file", "s", "", "Useful for debugging your specs on the command line, without having to make round trips to the server")
	}

	cmd.AddCommand(e2e.Cmd())
	cmd.AddCommand(devtool_releaser.Cmd())
	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/replicated")
		viper.AddConfigPath("/etc/sysconfig/")
		viper.SetConfigName("ship")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
