package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/util"
)

// ResolveRelease is a resolver that turns a target string into a release.
//
// A "target string" is something like
//
//   github.com/helm/charts/stable/nginx-ingress
//   replicated.app/cool-ci-tool?customer_id=...&installation_id=...
//   file:///home/bob/apps/ship.yaml
//   file:///home/luke/my-charts/proton-torpedoes
func (r *Resolver) ResolveRelease(ctx context.Context, upstream string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveRelease")
	r.ui.Info(fmt.Sprintf("Reading %s ...", upstream))

	r.ui.Info("Determining application type ...")
	applicationType, localPath, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", upstream)
	}
	debug.Log("event", "applicationType.resolve", "type", applicationType)
	r.ui.Info(fmt.Sprintf("Detected application type %s", applicationType))

	debug.Log("event", "upstream.Serialize", "for", localPath, "upstream", upstream)
	err = r.StateManager.SerializeUpstream(upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "write upstream")
	}

	switch applicationType {
	case "helm":
		if err := r.prepareDestPath(localPath, constants.HelmChartPath); err != nil {
			return nil, err
		}

		metadata, err := r.resolveMetadata(ctx, upstream, constants.HelmChartPath, applicationType)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve metadata for %s", constants.HelmChartPath)
		}

		renderedDest := fmt.Sprintf("%s.yaml", metadata.Name)
		defaultRelease := DefaultHelmRelease(constants.HelmChartPath, renderedDest)
		return r.resolveRelease(
			ctx,
			&defaultRelease,
			metadata,
			constants.HelmChartPath,
		)
	case "k8s":
		if err := r.prepareDestPath(localPath, constants.KustomizeBasePath); err != nil {
			return nil, err
		}

		metadata, err := r.resolveMetadata(ctx, upstream, constants.KustomizeBasePath, applicationType)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve metadata for %s", constants.KustomizeBasePath)
		}

		defaultRelease := DefaultRawRelease(constants.KustomizeBasePath)
		return r.resolveRelease(
			ctx,
			&defaultRelease,
			metadata,
			constants.KustomizeBasePath,
		)
	case "replicated.app":
		parsed, err := url.Parse(upstream)
		if err != nil {
			return nil, errors.Wrapf(err, "parse url %s", upstream)
		}
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)
		return r.AppResolver.ResolveAppRelease(ctx, selector)
	}

	return nil, errors.Errorf("unknown application type %q for upstream %q", applicationType, upstream)
}

// read the content sha without writing anything to state
func (r *Resolver) ReadContentSHAForWatch(ctx context.Context, upstream string) (string, error) {

	debug := level.Debug(log.With(r.Logger, "method", "ReadContentSHAForWatch"))
	debug.Log("event", "fetch latest chart")
	appType, localPath, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return "", errors.Wrapf(err, "resolve app type for %s", upstream)
	}
	debug.Log("event", "apptype.inspect", "type", appType, "localPath", localPath)

	// In this switch we only want to read the metadata without persisting anything to state,
	// and there doesn't seem to be a good way to evolve that abstraction cleanly from what we have, at least not just yet
	switch appType {

	case "helm":
		metadata, err := r.ResolveBaseMetadata(upstream, localPath)
		if err != nil {
			return "", errors.Wrapf(err, "resolve metadata and content sha for %s", upstream)
		}
		return metadata.ContentSHA, nil

	case "k8s":
		metadata, err := r.ResolveBaseMetadata(upstream, localPath)
		if err != nil {
			return "", errors.Wrapf(err, "resolve metadata and content sha for %s", upstream)
		}
		return metadata.ContentSHA, nil

	case "replicated.app":
		parsed, err := url.Parse(upstream)
		if err != nil {
			return "", errors.Wrapf(err, "parse url %s", upstream)
		}
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)

		release, err := r.AppResolver.FetchRelease(ctx, selector)
		if err != nil {
			return "", errors.Wrap(err, "fetch release")
		}

		return fmt.Sprintf("%x", sha256.Sum256([]byte(release.Spec))), nil
	}

	return "", errors.Errorf("Could not determine application type of upstream %s", upstream)
}

func (r *Resolver) prepareDestPath(localPath, destPath string) error {
	debug := log.With(level.Debug(r.Logger), "method", "prepareDestPath")

	if r.Viper.GetBool("rm-asset-dest") {
		err := r.FS.RemoveAll(destPath)
		if err != nil {
			return errors.Wrapf(err, "remove asset dest %s", destPath)
		}
	}

	err := util.BailIfPresent(r.FS, destPath, debug)
	if err != nil {
		return errors.Wrapf(err, "backup %s", destPath)
	}

	err = r.FS.Rename(localPath, destPath)
	if err != nil {
		return errors.Wrapf(err, "copy %s to %s", localPath, destPath)
	}

	return nil
}

func (r *Resolver) resolveRelease(
	ctx context.Context,
	defaultSpec *api.Spec,
	metadata *api.ShipAppMetadata,
	destPath string,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveRelease")

	debug.Log("event", "check upstream for ship.yaml")
	spec, err := r.maybeGetShipYAML(ctx, destPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", destPath)
	}

	if spec == nil {
		debug.Log("event", "no ship.yaml")
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
