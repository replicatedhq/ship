package cli

import (
	"fmt"
	"os"

	"github.com/replicatedhq/ship/pkg/constants"

	"strings"

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
		Long:  `ship allows for configuring and updating third party application in modern pipelines (gitops).`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			version.Init()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}
	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/replicated/ship.yaml)")
	cmd.PersistentFlags().String("log-level", "off", "Log level")

	// optional
	cmd.PersistentFlags().IntP("api-port", "p", 8800, "port to start the API server on.")
	cmd.PersistentFlags().BoolP("headless", "", false, "run ship in headless mode")
	cmd.PersistentFlags().Bool("no-open", false, "skip opening the ship console in the default browser--does not disable the UI, has no effect if `headless` is set to true.")
	cmd.PersistentFlags().String("state-file", "", "path to the state file to read from, defaults to "+constants.StatePath)
	cmd.PersistentFlags().String("resource-type", "", "upstream application resource type")

	cmd.AddCommand(Init())
	cmd.AddCommand(Watch())
	cmd.AddCommand(Update())
	cmd.AddCommand(App())
	cmd.AddCommand(Version())
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
