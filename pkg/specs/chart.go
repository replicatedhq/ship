package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"gopkg.in/yaml.v2"
)

func DefaultHelmRelease(chartPath string) api.Spec {
	return api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{
				{
					Helm: &api.HelmAsset{
						AssetShared: api.AssetShared{
							Dest: constants.KustomizeBasePath,
						},
						Local: &api.LocalHelmOpts{
							ChartRoot: chartPath,
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
						Root: ".",
					},
				},
				{
					KustomizeIntro: &api.KustomizeIntro{
						StepShared: api.StepShared{
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						BasePath: constants.KustomizeBasePath,
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
	}
}

func DefaultRawRelease(basePath string) api.Spec {
	return api.Spec{
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
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						BasePath: basePath,
						Dest:     path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:          "kustomize",
							Invalidates: []string{"diff"},
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
	}
}

func (r *Resolver) resolveMetadata(ctx context.Context, upstream, localPath string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveHelmMetadata"))

	baseMetadata, err := r.ResolveBaseMetadata(upstream, localPath)
	if err != nil {
		return nil, errors.Wrap(err, "resolve base metadata")
	}

	localChartPath := filepath.Join(localPath, "Chart.yaml")

	exists, err := r.FS.Exists(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}
	if !exists {
		return baseMetadata, nil
	}

	debug.Log("phase", "read-chart", "from", localChartPath)
	chart, err := r.FS.ReadFile(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}

	debug.Log("phase", "unmarshal-chart.yaml")
	if err := yaml.Unmarshal(chart, &baseMetadata); err != nil {
		return nil, err
	}

	return baseMetadata, nil
}

// ResolveBaseMetadata resolves URL, ContentSHA, and Readme for the resource
func (r *Resolver) ResolveBaseMetadata(upstream string, localPath string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveBaseMetaData"))
	var md api.ShipAppMetadata
	md.URL = upstream
	debug.Log("event", "upstream.Serialize", "for", localPath, "upstream", upstream)
	err := r.StateManager.SerializeUpstream(upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "write upstream")
	}
	debug.Log("phase", "calculate-sha", "for", localPath)
	contentSHA, err := r.shaSummer(r, localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "calculate chart sha")
	}
	md.ContentSHA = contentSHA

	err = r.StateManager.SerializeContentSHA(contentSHA)
	if err != nil {
		return nil, errors.Wrap(err, "write content sha")
	}

	localReadmePath := filepath.Join(localPath, "README.md")
	debug.Log("phase", "read-readme", "from", localReadmePath)
	readme, err := r.FS.ReadFile(localReadmePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "read file from %s", localReadmePath)
		}
	}
	if readme != nil {
		md.Readme = string(readme)
	} else {
		// TODO default README better
		md.Readme = fmt.Sprintf(`
Deployment Generator
===========================

This is a deployment generator for

    ship init %s

Sources for your app have been generated at %s. This installer will walk you through
customizing these resources and preparing them to deploy to your infrastructure.
`, upstream, localPath)
	}
	return &md, nil
}

func (r *Resolver) maybeGetShipYAML(ctx context.Context, localPath string) (*api.Spec, error) {
	localReleasePaths := []string{
		filepath.Join(localPath, "ship.yaml"),
		filepath.Join(localPath, "ship.yml"),
	}

	r.ui.Info("Looking for ship.yaml ...")

	for _, shipYAMLPath := range localReleasePaths {
		upstreamShipYAMLExists, err := r.FS.Exists(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "check file %s exists", shipYAMLPath)
		}

		if !upstreamShipYAMLExists {
			continue
		}
		upstreamRelease, err := r.FS.ReadFile(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "read file from %s", shipYAMLPath)
		}
		var spec api.Spec
		if err := yaml.UnmarshalStrict(upstreamRelease, &spec); err != nil {
			level.Debug(r.Logger).Log("event", "release.unmarshal.fail", "error", err)
			return nil, errors.Wrapf(err, "unmarshal ship.yaml")
		}
		return &spec, nil
	}

	return nil, nil
}

type shaSummer func(*Resolver, string) (string, error)

func (r *Resolver) calculateContentSHA(root string) (string, error) {
	var contents []byte
	err := r.FS.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "fs walk")
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fileContents, err := r.FS.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read file")
		}

		contents = append(contents, fileContents...)
		return nil
	})

	if err != nil {
		return "", errors.Wrapf(err, "calculate content sha")
	}

	return fmt.Sprintf("%x", sha256.Sum256(contents)), nil
}
