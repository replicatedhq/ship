package ship

import (
	"context"

	dockercli "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/fs"
	"github.com/replicatedcom/ship/pkg/lifecycle"
	"github.com/replicatedcom/ship/pkg/lifecycle/message"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/state"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/replicatedcom/ship/pkg/specs"
	"github.com/replicatedcom/ship/pkg/templates"
	"github.com/replicatedcom/ship/pkg/ui"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

func buildInjector() (*dig.Container, error) {
	providers := []interface{}{

		viper.GetViper,
		logger.FromViper,
		ui.FromViper,
		fs.FromViper,

		templates.NewBuilderBuilder,
		message.NewMessenger,
		config.NewDaemon,
		config.NewRenderer,
		config.NewHeadedDaemon,
		config.NewHeadlessDaemon,
		config.NewResolver,
		state.NewManager,
		planner.NewPlanner,
		render.NewRenderer,
		specs.NewResolver,
		specs.NewGraphqlClient,
		lifecycle.NewRunner,

		docker.URLResolverFromViper,
		docker.SaverFromViper,
		dockercli.NewEnvClient,

		NewShip,
	}

	container := dig.New()

	for _, provider := range providers {
		err := container.Provide(provider)
		if err != nil {
			return nil, errors.Wrap(err, "register providers")
		}
	}

	return container, nil
}

func Get() (*Ship, error) {

	injector, err := buildInjector()
	if err != nil {
		return nil, errors.Wrap(err, "build injector")
	}

	var ship *Ship

	// we return nil below , so the error will only ever be a construction error
	errorWhenConstructingShip := injector.Invoke(func(s *Ship) error {
		ship = s
		return nil
	})

	if errorWhenConstructingShip != nil {
		return nil, errors.Wrap(err, "resolve dependencies")
	}
	return ship, nil
}

func RunE(ctx context.Context) error {
	s, err := Get()
	if err != nil {
		return err
	}
	s.ExecuteAndMaybeExit(ctx)
	return nil
}
