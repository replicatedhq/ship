package cli

import (
	"fmt"
	"os"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ship",
		Short:         "manage and serve on-prem ship data",
		Long:          `ship allows for configuring and updating third party application in modern pipelines (gitops).`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
			os.Exit(1)
		},
		// I think its okay to use real OS filesystem commands instead of afero here,
		// since I think cobra lives outside the scope of dig injection/unit testing.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var multiErr *multierror.Error
			multiErr = multierror.Append(multiErr, os.RemoveAll(constants.ShipPathInternalTmp))
			multiErr = multierror.Append(multiErr, os.MkdirAll(constants.ShipPathInternalTmp, 0755))
			return multiErr.ErrorOrNil()

		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			var multiErr *multierror.Error
			multiErr = multierror.Append(multiErr, os.RemoveAll(constants.ShipPathInternalTmp))
			// if we got here, it means we finished successfully, so remove the internal debug log file
			multiErr = multierror.Append(multiErr, os.RemoveAll(constants.ShipPathInternalLog))
			return multiErr.ErrorOrNil()
		},
	}
	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/replicated/ship.yaml)")
	cmd.PersistentFlags().String("log-level", "off", "Log level")

	cmd.PersistentFlags().IntP("api-port", "p", 8800, "port to start the API server on.")
	cmd.PersistentFlags().Bool("no-open", false, "skip opening the ship console in the default browser--does not disable the UI, has no effect if `headless` is set to true.")

	cmd.PersistentFlags().StringP("customer-endpoint", "e", "https://pg.replicated.com/graphql", "Upstream application spec server address")

	cmd.PersistentFlags().BoolP("headless", "", false, "run ship in headless mode")
	// TODO remove me, just always set this to true
	cmd.PersistentFlags().BoolP("navcycle", "", true, "set to false to run ship in v1/non-navigable mode (deprecated)")

	cmd.PersistentFlags().String("state-from", "file", "type of resource to use when loading/saving state (currently supported values: 'file', 'secret', 'url'")
	cmd.PersistentFlags().String("state-file", "", fmt.Sprintf("path to the state file to read from, defaults to %s", constants.StatePath))
	cmd.PersistentFlags().String("secret-namespace", "default", "namespace containing the state secret")
	cmd.PersistentFlags().String("secret-name", "", "name of the secret to load state from")
	cmd.PersistentFlags().String("secret-key", "", "name of the key in the secret containing state")
	cmd.PersistentFlags().String("state-put-url", "", "the URL that will be used to store update state")
	cmd.PersistentFlags().String("state-get-url", "", "the URL that will be used to retrieve update state")

	cmd.PersistentFlags().String("upload-assets-to", "", "URL to upload assets to via HTTP PUT request. NOTE: this will cause the entire working directory to be uploaded to the specified URL, use with caution.")

	cmd.PersistentFlags().String("terraform-exec-path", "terraform", "Path to a terraform executable on the system.")
	cmd.PersistentFlags().Bool("terraform-apply-yes", false, "Automatically apply terraform steps in headless mode. By default, terraform will be skipped when ship is running in automation.")

	cmd.PersistentFlags().Bool("no-web", false, "Disable web assets")

	cmd.PersistentFlags().String("resource-type", "", "upstream application resource type")
	cmd.PersistentFlags().BoolP("prefer-git", "", false, "prefer the git protocol instead of using http apis")

	cmd.PersistentFlags().StringP("helm-values-file", "", "", "Optional file path to Values.yaml to be used when rendering Helm charts (only supported in headless mode)")

	cmd.PersistentFlags().Bool("no-outro", false, "skip outro step in Ship UI")

	cmd.PersistentFlags().Bool(constants.FilesInStateFlag, false, "persist files (helm values and defaults files, kustomize patches and resources) in the ship state in addition to .ship/helm and the overlays directory")

	cmd.AddCommand(Init())
	cmd.AddCommand(Watch())
	cmd.AddCommand(Update())
	cmd.AddCommand(Edit())
	cmd.AddCommand(Unfork())
	cmd.AddCommand(App())
	cmd.AddCommand(Version())

	_ = viper.BindPFlags(cmd.Flags())
	_ = viper.BindPFlags(cmd.PersistentFlags())
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
