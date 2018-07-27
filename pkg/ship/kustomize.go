package ship

import (
	"context"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
)

func (s *Ship) KustomizeAndMaybeExit(ctx context.Context) {
	if err := s.Kustomize(ctx); err != nil {
		s.ExitWithError(err)
	}
}

func (s *Ship) Kustomize(ctx context.Context) error {
	if s.IsKustomize && (s.Viper.GetString("raw") != "") {
		release := s.fakeKustomizeRawRelease()
		return s.execute(ctx, release, nil)
	}

	release := &api.Release{

		Spec: api.Spec{
			Lifecycle: api.Lifecycle{
				V1: []api.Step{
					{
						HelmIntro: &api.HelmIntro{},
					},
					{
						HelmValues: &api.HelmValues{},
					},
				},
			},
		},
	}
	helmChartPath := s.Viper.GetString("chart")
	helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), helmChartPath)
	release.Metadata.HelmChartMetadata = helmChartMetadata
	if err != nil {
		errors.Wrapf(err, "resolve helm metadata for %s", helmChartPath)
	}
	return s.execute(ctx, release, nil)
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
							BasePath: s.KustomizeRaw,
							Dest:     path.Join(constants.InstallerPrefix, "kustomized"),
						},
					},
					{
						Message: &api.Message{
							Contents: `
Assets are ready to deploy. You can run

    kubectl apply -f installer/rendered

to deploy the overlaid assets to your cluster.
						`},
					},
				},
			},
		},
	}

	return release
}
