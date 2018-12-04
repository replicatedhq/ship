package ship

import (
	"net/http"
	"time"

	dockercli "github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/filetree"
	"github.com/replicatedhq/ship/pkg/fs"
	"github.com/replicatedhq/ship/pkg/images"
	"github.com/replicatedhq/ship/pkg/lifecycle"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/headless"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/statusonly"
	"github.com/replicatedhq/ship/pkg/lifecycle/helmIntro"
	"github.com/replicatedhq/ship/pkg/lifecycle/helmValues"
	"github.com/replicatedhq/ship/pkg/lifecycle/kubectl"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomize"
	"github.com/replicatedhq/ship/pkg/lifecycle/kustomizeintro"
	"github.com/replicatedhq/ship/pkg/lifecycle/message"
	"github.com/replicatedhq/ship/pkg/lifecycle/render"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/amazoneks"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/azureaks"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/config/resolve"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/docker"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/dockerlayer"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/github"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/googlegke"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/helm"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/local"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/terraform"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/web"
	terraform2 "github.com/replicatedhq/ship/pkg/lifecycle/terraform"
	"github.com/replicatedhq/ship/pkg/lifecycle/terraform/tfplan"
	"github.com/replicatedhq/ship/pkg/lifecycle/unfork"
	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/ui"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

func buildInjector(v *viper.Viper) (*dig.Container, error) {

	providers := []interface{}{

		func() *http.Client { return http.DefaultClient },
		provide(v),
		clock,
		logger.New,
		ui.FromViper,
		fs.NewBaseFilesystem,
		daemon.WebUIFactoryFactory,
		filetree.NewLoader,
		templates.NewBuilderBuilder,
		specs.NewIDPatcher,
		apptype.NewInspector,
		util.NewAssetUploader,
		patch.NewShipPatcher,

		daemon.NewV1Router,
		resolve.NewRenderer,

		state.NewManager,
		planner.NewFactory,
		specs.NewResolver,
		replicatedapp.NewGraphqlClient,
		replicatedapp.NewAppResolver,
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
		helm.NewCommands,

		web.NewStep,

		github.NewRenderer,
		githubclient.NewGithubClient,

		terraform.NewRenderer,

		amazoneks.NewRenderer,

		googlegke.NewRenderer,

		azureaks.NewRenderer,

		local.NewRenderer,

		NewShip,
	}

	// we used to do
	//
	//     if viper.GetBool(...) { /* decide which interface implementation to return */ }
	//
	// in constructor methods that were passed to dig.New(), but now
	// need to switch on mode here to avoid circular dependencies *in the object graph*
	// (as opposed to in the source graph). Even though lifecycle doesn't depend
	// on source code that depends on daemon, the StepExecutor constructor still depends
	// on objects that depend on daemon, so in order to be able to use packages like
	//
	//  - lifeycle/message
	//  - lifeycle/kustomize
	//  - lifeycle/helmIntro
	//
	// in navigable mode,
	// we need to keep the *implementations that need an instance of daemon* out of the DI container
	//
	// Hopefully once everything is moved over to v2 this gets a lot simpler again.
	if viper.GetBool("headless") {
		providers = append(providers, headlessProviders()...)
	} else if viper.GetBool("navcycle") {
		providers = append(providers, navcycleProviders()...)
	} else {
		providers = append(providers, headedProviders()...)

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

// "headedless mode" is the standard execute-the-lifecycle-in-order mode of ship, that runs without UI or api server
// and is generally intended for CI/automation
func headlessProviders() []interface{} {
	return []interface{}{
		headless.NewHeadlessDaemon,
		helmIntro.NewHelmIntro,
		config.NewResolver,
		render.NewFactory,
		helmValues.NewHelmValues,
		kustomize.NewDaemonKustomizer,
		unfork.NewDaemonUnforker,
		terraform2.NewTerraformer,
		tfplan.NewPlanner,
		kubectl.NewKubectl,
		func(messenger message.CLIMessenger) lifecycle.Messenger { return &messenger },
		func(d daemontypes.Daemon) daemontypes.StatusReceiver { return d },
	}
}

// "headed mode" is the standard execute-the-lifecycle-in-order mode of ship, where steps manipulate
// the UI/API via a ShipDaemon implementing the daemon.Daemon interface
func headedProviders() []interface{} {
	return []interface{}{
		daemon.NewHeadedDaemon,
		helmIntro.NewHelmIntro,
		config.NewResolver,
		render.NewFactory,
		helmValues.NewHelmValues,
		kustomize.NewDaemonKustomizer,
		unfork.NewDaemonUnforker,
		terraform2.NewTerraformer,
		tfplan.NewPlanner,
		kubectl.NewKubectl,
		func(messenger message.DaemonMessenger) lifecycle.Messenger { return &messenger },
		func(d daemontypes.Daemon) daemontypes.StatusReceiver { return d },
	}
}

// "navigable mode" provides a new, v2-ish version of ship that provides browser navigation back
// and forth through the lifecycle, and uses runbook declarations of lifecycle dependencies to
// control execution ordering and workflows
func navcycleProviders() []interface{} {
	return []interface{}{
		daemon.NewHeadedDaemon,
		render.NoConfigRenderer,
		config.NewNoOpResolver,
		helmValues.NewDaemonlessHelmValues,
		kustomizeintro.NewKustomizeIntro,
		kustomize.NewDaemonlessKustomizer,
		unfork.NewDaemonlessUnforker,
		func(messenger message.DaemonlessMessenger) lifecycle.Messenger { return &messenger },
		func(intro helmIntro.DaemonlessHelmIntro) lifecycle.HelmIntro { return &intro },
		terraform2.NewDaemonlessTerraformer,
		tfplan.NewDaemonlessPlanner,
		kubectl.NewDaemonlessKubectl,
		// fake, we override it, this is janky, use a factory dex
		func() daemontypes.StatusReceiver { return &statusonly.StatusReceiver{} },
		daemon.NewV2Router,
	}
}

func Get(v *viper.Viper) (*Ship, error) {
	// who injects the injectors?
	debug := log.With(level.Debug(logger.New(v, fs.NewBaseFilesystem())), "component", "injector", "phase", "instance.get")

	debug.Log("event", "injector.build")
	injector, err := buildInjector(v)
	if err != nil {
		debug.Log("event", "injector.build.fail", "error", err)
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
		debug.Log("event", "injector.invoke.fail", "err", errorWhenConstructingShip)
		return nil, errors.Wrap(errorWhenConstructingShip, "resolve dependencies")
	}
	return ship, nil
}

func clock() func() time.Time {
	clock := func() time.Time {
		return time.Now()
	}
	return clock
}

func provide(v *viper.Viper) func() *viper.Viper {
	return func() *viper.Viper {
		return v
	}
}
