package specs

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
)

// A resolver turns a target string into a release.
//
// A "target string" is something like
//
//   github.com/helm/charts/stable/nginx-ingress
//   replicated.app/cool-ci-tool?customer_id=...&installation_id=...
//   file:///home/bob/apps/ship.yml
//   file:///home/luke/my-charts/deathstar_destroyer
func (r *Resolver) ResolveRelease(ctx context.Context, target string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveRelease")
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, errors.Wrapf(err, "parse url %s", target)

	}
	r.ui.Info(fmt.Sprintf("Reading %s ...", target))
	// todo fetch it

	r.ui.Info("Determining application type ...")
	applicationType := r.determineApplicationType(target)
	debug.Log("event", "applicationType.resolve", "type", applicationType)
	r.ui.Info(fmt.Sprintf("Detected application type %s", applicationType))

	switch applicationType {
	case "helm":
		return r.resolveChart(ctx, target)
	case "replicated.app":
		selector := (&Selector{}).unmarshalFrom(parsed)
		return r.ResolveAppRelease(ctx, selector)
	}

	return nil, errors.Errorf("unknown application type %q for target %q", applicationType, target)
}

func (r *Resolver) resolveChart(ctx context.Context, target string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveChart")

	chartRepoURL := r.Viper.GetString("chart-repo-url")
	chartVersion := r.Viper.GetString("chart-version")
	helmChartMetadata, err := r.ResolveChartMetadata(context.Background(), target, chartRepoURL, chartVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve helm metadata for %s", target)
	}

	// serialize the ChartURL to disk. First step in creating a state file
	r.StateManager.SerializeChartURL(target)

	// persist helm options
	err = r.StateManager.SaveHelmOpts(chartRepoURL, chartVersion)
	if err != nil {
		return nil, errors.Wrap(err, "write helm opts")
	}

	debug.Log("event", "check upstream release")
	spec, err := r.ResolveChartReleaseSpec(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve chart release for %s", filepath.Join(constants.KustomizeHelmPath, "ship.yaml"))
	}

	debug.Log("event", "build helm release")
	err = r.StateManager.SerializeContentSHA(helmChartMetadata.ContentSHA)
	if err != nil {
		return nil, errors.Wrap(err, "write content sha")
	}

	return &api.Release{
		Metadata: api.ReleaseMetadata{
			HelmChartMetadata: helmChartMetadata,
		},
		Spec: spec,
	}, nil
}

func (r *Resolver) determineApplicationType(target string) string {
	// hack hack hack
	isReplicatedApp := strings.HasPrefix(target, "replicated.app") ||
		strings.HasPrefix(target, "staging.replicated.app") ||
		strings.HasPrefix(target, "local.replicated.app")

	applicationType := "helm"
	if isReplicatedApp {
		applicationType = "replicated.app"

	}

	// todo more types
	return applicationType
}
