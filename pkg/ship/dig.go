package ship

import (
	"context"

	dockercli "github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/fs"
	"github.com/replicatedhq/ship/pkg/images"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/lifecycle/message"
	"github.com/replicatedhq/ship/pkg/lifecycle/render"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/dockerlayer"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/helm"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/terraform"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/web"
	terraform2 "github.com/replicatedhq/ship/pkg/lifecycle/terraform"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

func buildInjector() (*dig.Container, error) {
	providers := []interface{}{

		viper.GetViper,
		logger.FromViper,
		ui.FromViper,
		fs.FromViper,
		daemon.WebUIFactoryFactory,

		templates.NewBuilderBuilder,
		message.NewMessenger,
		config.NewDaemon,
		daemon.NewHeadedDaemon,
		daemon.NewHeadlessDaemon,
		config.NewResolver,
		resolve.NewRenderer,
		terraform2.NewTerraformer,
		kustomize.NewKustomizer,
		tfplan.NewPlanner,

		state.NewManager,
		planner.NewPlanner,
		render.NewRenderer,
		specs.NewResolver,
		specs.NewGraphqlClient,
		lifecycle.NewRunner,

		inline.NewRenderer,

		images.URLResolverFromViper,
		images.NewImageSaver,

		docker.NewStep,

		dockercli.NewEnvClient,

		dockerlayer.NewUnpacker,
		dockerlayer.TarArchiver,

		helm.NewRenderer,
		helm.NewFetcher,
		helm.NewTemplater,

		web.NewStep,

		github.NewRenderer,

		terraform.NewRenderer,

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
	// who injects the injectors?
	debug := log.With(level.Debug(logger.FromViper(viper.GetViper())), "component", "injector", "phase", "instance.get")

	debug.Log("event", "injector.build")
	injector, err := buildInjector()
	if err != nil {
		debug.Log("event", "injector.build.fail")
		return nil, errors.Wrap(err, "build injector")
	}

	var ship *Ship

	// we return nil below , so the error will only ever be a construction error
	debug.Log("event", "injector.invoke")
	errorWhenConstructingShip := injector.Invoke(func(s *Ship) {
		debug.Log("event", "injector.invoke.resolve")
		ship = s
	})

	if errorWhenConstructingShip != nil {
		debug.Log("event", "injector.invoke.fail")
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
