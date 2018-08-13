package ship

import (
	"context"
	"path"
	"time"

	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
)

func (s *Ship) InitAndMaybeExit(ctx context.Context) {
	if err := s.Init(ctx); err != nil {
		if err.Error() == constants.ShouldUseUpdate {
			s.ExitWithWarn(err)
		}
		s.ExitWithError(err)
	}
}

func (s *Ship) WatchAndExit(ctx context.Context) {
	if err := s.Watch(ctx); err != nil {
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

	s.Daemon.SetProgress(daemontypes.StringProgress("kustomize", `loading state from `+constants.StatePath))

	if _, noExistingState := existingState.(state.Empty); noExistingState {
		debug.Log("event", "state.missing")
		return errors.New(`No state file found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
	}

	debug.Log("event", "read.chartURL")
	helmChartPath := existingState.CurrentChartURL()
	if helmChartPath == "" {
		return errors.New(`No helm chart URL found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
	}

	debug.Log("event", "fetch latest chart")
	s.Daemon.SetProgress(daemontypes.StringProgress("kustomize", `downloading latest from upstream `+helmChartPath))
	helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), string(helmChartPath))
	if err != nil {
		return errors.Wrapf(err, "resolve helm chart metadata for %s", helmChartPath)
	}

	release := s.buildRelease(helmChartMetadata)

	return s.execute(ctx, release, nil, true)
}

func (s *Ship) Watch(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "watch"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	for {
		existingState, err := s.State.TryLoad()

		if _, noExistingState := existingState.(state.Empty); noExistingState {
			debug.Log("event", "state.missing")
			return errors.New(`No state file found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		debug.Log("event", "read.chartURL")
		helmChartPath := existingState.CurrentChartURL()
		if helmChartPath == "" {
			return errors.New(`No current chart url found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		debug.Log("event", "read.lastSHA")
		lastSHA := existingState.Versioned().V1.ContentSHA
		if lastSHA == "" {
			return errors.New(`No current SHA found at ` + s.Viper.GetString("state-file") + `, please run "ship init"`)
		}

		debug.Log("event", "fetch latest chart")
		helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), string(helmChartPath))
		if err != nil {
			return errors.Wrapf(err, "resolve helm chart metadata for %s", helmChartPath)
		}

		if helmChartMetadata.ContentSHA != existingState.Versioned().V1.ContentSHA {
			debug.Log("event", "new sha")
			return nil
		}

		time.Sleep(time.Minute * 5) // todo flag
	}
}

func (s *Ship) Init(ctx context.Context) error {
	debug := level.Debug(log.With(s.Logger, "method", "init"))
	ctx, cancelFunc := context.WithCancel(ctx)
	defer s.Shutdown(cancelFunc)

	if s.Viper.GetString("raw") != "" {
		release := s.fakeKustomizeRawRelease()
		return s.execute(ctx, release, nil, true)
	}

	// does a state file exist on disk?
	if s.stateFileExists(ctx) && !s.Viper.GetBool("rm-state") {
		debug.Log("event", "state.exists")

		useUpdate, err := s.UI.Ask(`
An existing .ship directory was found. If you are trying to update this application, run "ship update".
Continuing will delete this state, would you like to continue? There is no undo. (y/N)
`)
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

	// serialize the ChartURL to disk. First step in creating a state file
	s.State.SerializeChartURL(helmChartPath)

	release := s.buildRelease(helmChartMetadata)
	release.Spec.Lifecycle = s.IDPatcher.EnsureAllStepsHaveUniqueIDs(release.Spec.Lifecycle)

	s.State.SerializeContentSHA(helmChartMetadata.ContentSHA)

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
						KustomizeIntro: &api.KustomizeIntro{
							StepShared: api.StepShared{
								ID: "kustomize",
							},
						},
					},
					{
						Kustomize: &api.Kustomize{
							BasePath: s.KustomizeRaw,
							Dest:     path.Join("overlays", "ship"),
							StepShared: api.StepShared{
								ID:          "kustomize",
								Invalidates: []string{"diff"},
							},
						},
					},
					{
						KustomizeDiff: &api.KustomizeDiff{
							StepShared: api.StepShared{
								ID:       "kustomize",
								Requires: []string{"kustomize"},
							},
						},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{
								ID: "outro",
							},
							Contents: `
Assets are ready to deploy. You can run

    kustomize build overlays/ship | kubectl apply -f -

to deploy the overlaid assets to your cluster.
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
								Dest: constants.RenderedHelmPath,
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
								ID: "intro",
							},
						},
					},
					{
						HelmValues: &api.HelmValues{
							StepShared: api.StepShared{
								ID:          "values",
								Requires:    []string{"intro"},
								Invalidates: []string{"render"},
							},
						},
					},
					{
						Render: &api.Render{
							StepShared: api.StepShared{
								ID:       "render",
								Requires: []string{"values"},
							},
						},
					},
					{
						Kustomize: &api.Kustomize{
							BasePath: constants.RenderedHelmPath,
							Dest:     path.Join("overlays", "ship"),
							StepShared: api.StepShared{
								ID:       "kustomize",
								Requires: []string{"render"},
							},
						},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{
								ID:       "outro",
								Requires: []string{"kustomize"},
							},
							Contents: `
Assets are ready to deploy. You can run

    kustomize build overlays/ship | kubectl apply -f -

to deploy the overlaid assets to your cluster.
						`},
					},
				},
			},
		},
	}

	return release
}
