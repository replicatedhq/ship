package ship

import (
	"context"

	"fmt"
	"os"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/lifecycle"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/replicatedcom/ship/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Ship configures an application
type Ship struct {
	Viper *viper.Viper

	Logger kitlog.Logger

	Port           int
	CustomerID     string
	InstallationID string
	PlanOnly       bool

	Resolver   *specs.Resolver
	StudioFile string
	Client     *specs.GraphQLClient
	UI         cli.Ui
}

// FromViper gets an instance using viper to pull config
func FromViper(v *viper.Viper) (*Ship, error) {

	resolver, err := specs.ResolverFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "get spec resolver")
	}

	graphql, err := specs.GraphQLClientFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "get graphql client")
	}

	return &Ship{
		Viper: v,

		Logger:   logger.FromViper(v),
		Resolver: resolver,
		Client:   graphql,

		Port:           v.GetInt("port"),
		CustomerID:     v.GetString("customer-id"),
		InstallationID: v.GetString("installation_id"),
		StudioFile:     v.GetString("studio-file"),

		UI: &cli.ColoredUi{
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
	}, nil
}

// Execute starts ship
func (d *Ship) Execute(ctx context.Context) error {
	debug := level.Debug(kitlog.With(d.Logger, "method", "execute"))

	debug.Log("method", "configure", "phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
		"customer-id", d.CustomerID,
		"installation_id", d.InstallationID,
		"plan_only", d.PlanOnly,
		"studio-file", d.StudioFile,
		"studio", specs.AllowInlineSpecs,
		"port", d.Port,
	)

	debug.Log("phase", "validate-inputs")

	if d.StudioFile != "" && !specs.AllowInlineSpecs {
		debug.Log("phase", "load-specs", "error", "unsupported studio-file")
		return errors.New("unsupported configuration: studio-file")

	}

	spec, err := d.Resolver.ResolveSpecs(ctx, d.CustomerID)
	if err != nil {
		return errors.Wrap(err, "resolve specs")
	}

	// execute lifecycle
	lc := &lifecycle.Runner{
		CustomerID:     d.CustomerID,
		InstallationID: d.InstallationID,
		GraphQLClient:  d.Client,
		UI:             d.UI,
		Logger:         d.Logger,
		Spec:           spec,
		Fs:             afero.Afero{Fs: afero.NewOsFs()},
		Viper:          d.Viper,
	}

	return errors.Wrap(lc.Run(ctx), "run lifecycle")
}

// OnError should be called by the parent cobra commands if something goes wrong.
func (d *Ship) OnError(err error) {
	d.UI.Error(fmt.Sprintf("There was an unexpected error! %+v", err))
	d.UI.Output("")
	// TODO this should probably be part of lifecycle
	d.UI.Info("There was an error configuring the application. Please re-run with --log-level=debug and include the output in any support inquiries.")
}
