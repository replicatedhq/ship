package specs

import (
	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// A Resolver resolves specs
type Resolver struct {
	Logger       log.Logger
	Client       *replicatedapp.GraphQLClient
	StateManager state.Manager
	FS           afero.Afero
	AppResolver  replicatedapp.Resolver

	ui               cli.Ui
	appTypeInspector apptype.Inspector
	shaSummer        shaSummer

	Viper *viper.Viper
}

// NewResolver builds a resolver from a Viper instance

func NewResolver(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	graphql *replicatedapp.GraphQLClient,
	stateManager state.Manager,
	ui cli.Ui,
	determiner apptype.Inspector,
	appresolver replicatedapp.Resolver,
) *Resolver {
	return &Resolver{
		Logger:           logger,
		Client:           graphql,
		StateManager:     stateManager,
		FS:               fs,
		Viper:            v,
		ui:               ui,
		appTypeInspector: determiner,
		shaSummer: func(resolver *Resolver, s string) (string, error) {
			return resolver.calculateContentSHA(s)
		},
		AppResolver: appresolver,
	}
}
