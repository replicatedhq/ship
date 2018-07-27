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

	helmChartPath := s.Viper.GetString("chart")
	helmChartMetadata, err := s.Resolver.ResolveChartMetadata(context.Background(), helmChartPath)
	if err != nil {
		errors.Wrapf(err, "resolve helm metadata for %s", helmChartPath)
	}

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
						HelmIntro: &api.HelmIntro{},
					},
					{
						HelmValues: &api.HelmValues{},
					},
					{
						Render: &api.Render{},
					},
					{
						Kustomize: &api.Kustomize{
							BasePath: path.Join(constants.InstallerPrefix, helmChartMetadata.Name),
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
