package cli

import (
	"fmt"
	"os"

	"strings"

	"context"

	"github.com/replicatedhq/ship/pkg/cli/devtoolreleaser"
	"github.com/replicatedhq/ship/pkg/e2e"
	"github.com/replicatedhq/ship/pkg/ship"
	"github.com/replicatedhq/ship/pkg/version"
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
			return ship.RunE(context.Background())
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
	cmd.PersistentFlags().IntP("api-port", "p", 8880, "port to start the API server on.")
	cmd.PersistentFlags().BoolP("headless", "", false, "run ship in headless mode")

	cmd.PersistentFlags().String("studio-file", "", "Useful for debugging your specs on the command line, without having to make round trips to the server")
	cmd.PersistentFlags().String("state-file", "", "path to the state file to read from, defaults to .ship/state.json")
	cmd.PersistentFlags().String("studio-channel-name", "", "Useful for debugging your specs on the command line, without having to make round trips to the server")
	cmd.PersistentFlags().String("studio-channel-icon", "", "Useful for debugging your specs on the command line, without having to make round trips to the server")
	cmd.PersistentFlags().Bool("terraform-yes", false, "Automatically answer \"yes\" to all terraform prompts")

	cmd.AddCommand(e2e.Cmd())
	cmd.AddCommand(devtoolreleaser.Cmd())
	cmd.AddCommand(Kustomize())
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
