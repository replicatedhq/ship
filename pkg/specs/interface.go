package specs

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
)

// A resolver turns a target string into a release.
//
// A "target string" is something like
//
//   github.com/helm/charts/stable/nginx-ingress
//   replicated.app/cool-ci-tool?customer_id=...&installation_id=...
//   file:///home/bob/apps/ship.yaml
//   file:///home/luke/my-charts/proton-torpedoes
func (r *Resolver) ResolveRelease(ctx context.Context, upstream string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveRelease")
	parsed, err := url.Parse(upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "parse url %s", upstream)

	}
	r.ui.Info(fmt.Sprintf("Reading %s ...", upstream))

	r.ui.Info("Determining application type ...")
	applicationType, localPath, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", upstream)
	}
	debug.Log("event", "applicationType.resolve", "type", applicationType)
	r.ui.Info(fmt.Sprintf("Detected application type %s", applicationType))

	switch applicationType {
	case "helm":
		defaultRelease := DefaultHelmRelease(localPath)
		return r.resolveRelease(ctx, upstream, localPath, &defaultRelease)
	case "k8s":
		defaultRelease := DefaultRawRelease(localPath)
		return r.resolveRelease(ctx, upstream, localPath, &defaultRelease)
	case "replicated.app":
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)
		return r.AppResolver.ResolveAppRelease(ctx, selector)
	}

	return nil, errors.Errorf("unknown application type %q for upstream %q", applicationType, upstream)
}

func (r *Resolver) resolveRelease(
	ctx context.Context,
	upstream,
	localPath string,
	defaultSpec *api.Spec,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveChart")

	metadata, err := r.resolveMetadata(context.Background(), upstream, localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", localPath)
	}

	debug.Log("event", "check upstream for ship.yaml")
	spec, err := r.maybeGetShipYAML(ctx, localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", localPath)
	}

	if spec == nil {
		debug.Log("event", "no helm release")
		r.ui.Info("ship.yaml not found in upstream, generating default lifecycle for application ...")
		spec = defaultSpec
	}

	return &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
		},
		Spec: *spec,
	}, nil
}
