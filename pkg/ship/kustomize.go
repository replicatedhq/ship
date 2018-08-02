package ship

import (
	"context"
	"path"

	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/viper"
)

func (s *Ship) InitAndMaybeExit(ctx context.Context) {
	if err := s.Init(ctx); err != nil {
		if err.Error() == constants.ShouldUseUpdate {
			s.ExitWithWarn(err)
		}
		s.ExitWithError(err)
	}
}
func (s *Ship) UpdateAndMaybeExit(ctx context.Context) {
	if err := s.Update(ctx); err != nil {
		s.ExitWithError(err)
	}
}

func (s *Ship) stateFileExists(ctx context.Context) bool {
	debug := level.Debug(log.With(s.Logger, "method", "stateFileExists"))

	existingState, err := s.State.TryLoad()
	if err != nil {
		debug.Log("event", "tryLoad.fail")
		return false
	}
	_, noExistingState := existingState.(state.Empty)

	return !noExistingState
}

func (s *Ship) Update(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "update"))

	// does a state file exist on disk?
	existingState, err := s.State.TryLoad()

	if _, noExistingState := existingState.(state.Empty); noExistingState {
		debug.Log("event", "state.missing")
		return errors.New(`No state file found at ` + constants.StatePath + `, please run "ship init"`)
	}

	debug.Log("event", "read.chartURL")
	helmChartPath := existingState.CurrentChartURL()
	if helmChartPath == "" {
		return errors.New(`No helm chart URL found at ` + constants.StatePath + `, please run "ship init"`)
	}

	debug.Log("event", "fetch latest chart")
	helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), string(helmChartPath))
	if err != nil {
		return errors.Wrapf(err, "resolve helm chart metadata for %s", helmChartPath)
	}

	release := s.buildRelease(helmChartMetadata)

	// log for compile, will adjust later
	debug.Log("event", "build release", "release", release)

	// default to headless if user doesn't set --headed=true
	if viper.GetBool("headed") {
		viper.Set("headless", false)
	} else {
		viper.Set("headless", true)
	}

	// TODO IMPLEMENT
	return errors.New("Not implemented")
}

func (s *Ship) Init(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "init"))

	if s.Viper.GetString("raw") != "" {
		release := s.fakeKustomizeRawRelease()
		return s.execute(ctx, release, nil, true)
	}

	// does a state file exist on disk?
	if s.stateFileExists(ctx) {
		debug.Log("event", "state.exists")

		useUpdate, err := s.UI.Ask(`State file found at ` + constants.StatePath + `, do you want to start from scratch? (y/N) `)
		if err != nil {
			return err
		}
		useUpdate = strings.ToLower(strings.Trim(useUpdate, " \r\n"))

		if strings.Compare(useUpdate, "y") == 0 {
			// remove state.json and start from scratch
			if err := s.State.RemoveStateFile(); err != nil {
				return err
			}
		} else {
			// exit and use 'ship update'
			return errors.New(constants.ShouldUseUpdate)
		}
	}

	helmChartPath := s.Viper.GetString("chart")
	helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), helmChartPath)
	if err != nil {
		return errors.Wrapf(err, "resolve helm metadata for %s", helmChartPath)
	}

	release := s.buildRelease(helmChartMetadata)
	patchedLifecycle := s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)
	release.Spec.Lifecycle = patchedLifecycle

	return s.execute(ctx, release, nil, true)
}

func (s *Ship) fakeKustomizeRawRelease() *api.Release {
	release := &api.Release{
		Spec: api.Spec{
			Assets: api.Assets{
				V1: []api.Asset{},
			},
			Config: api.Config{
				V1: []libyaml.ConfigGroup{},
			},
			Lifecycle: api.Lifecycle{
				V1: []api.Step{
					{
						Kustomize: &api.Kustomize{
							StepShared: api.StepShared{Description: "Customize your yaml"},
							BasePath:   s.KustomizeRaw,
							Dest:       path.Join(constants.InstallerPrefix, "kustomized"),
						},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{Description: "Finalize"},
							Contents: `
Assets are ready to deploy. If you have [kustomize](https://github.com/kubernetes-sigs/kustomize) installed,
You can run

    kustomize build overlays/ship | kubectl apply -f -

to deploy the assets to your cluster.
						`},
					},
				},
			},
		},
	}

	return release
}

func (s *Ship) buildRelease(helmChartMetadata api.HelmChartMetadata) *api.Release {

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			HelmChartMetadata: helmChartMetadata,
		},
		Spec: api.Spec{
			Assets: api.Assets{
				V1: []api.Asset{
					{
						Helm: &api.HelmAsset{
							AssetShared: api.AssetShared{
								Dest: ".",
							},
							Local: &api.LocalHelmOpts{
								ChartRoot: constants.KustomizeHelmPath,
							},
							HelmOpts: []string{
								"--values",
								path.Join(constants.TempHelmValuesPath, "values.yaml"),
							},
						},
					},
				},
			},
			Lifecycle: api.Lifecycle{
				V1: []api.Step{
					{
						HelmIntro: &api.HelmIntro{
							StepShared: api.StepShared{
								ID:          "intro",
								Description: "Introduction to Chart",
							},
						},
					},
					{
						HelmValues: &api.HelmValues{
							StepShared: api.StepShared{
								ID:          "values",
								Description: "Customize Helm Values",
							},
						},
					},
					{
						Render: &api.Render{
							StepShared: api.StepShared{Description: "Render Helm Chart"},
						},
					},
					{
						KustomizeIntro: &api.KustomizeIntro{
							StepShared: api.StepShared{
								Description: "Kustomize Intro",
								ID:          "kustomize-intro",
							},
						},
					},
					{
						Kustomize: &api.Kustomize{
							StepShared: api.StepShared{
								Description: "Build Kustomize Patches",
								ID:          "kustomize",
							},
							BasePath: path.Join(constants.InstallerPrefix, helmChartMetadata.Name),
							Dest:     path.Join(constants.InstallerPrefix, "kustomized"),
						},
					},
					{
						KustomizeDiff: &api.KustomizeDiff{
							StepShared: api.StepShared{
								Description: "Review Kustomize Patches",
								ID:          "kustomize-diff",
							},
							BasePath: path.Join(constants.InstallerPrefix, helmChartMetadata.Name),
							Dest:     path.Join(constants.InstallerPrefix, "kustomized"),
						},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{
								Description: "Next Steps",
								ID:          "kustomize-diff",
							},
							Contents: `
Assets are ready to deploy. If you have [kustomize](https://github.com/kubernetes-sigs/kustomize) installed,
You can run

    kustomize build overlays/ship | kubectl apply -f -

to deploy the assets to your cluster.
						`},
					},
				},
			},
		},
	}

	return release
}
