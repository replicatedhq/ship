package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/util"
)

// A resolver turns a target string into a release.
//
// A "target string" is something like
//
//   github.com/helm/charts/stable/nginx-ingress
//   replicated.app/cool-ci-tool?customer_id=...&installation_id=...
//   file::/home/bob/apps/ship.yaml
//   file::/home/luke/my-charts/proton-torpedoes
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

	debug.Log("event", "versionedUpstream.resolve", "type", applicationType)
	versionedUpstream, err := r.maybeCreateVersionedUpstream(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "resolve versioned upstream")
	}

	debug.Log("event", "upstream.Serialize", "for", localPath, "upstream", versionedUpstream)
	err = r.StateManager.SerializeUpstream(versionedUpstream)
	if err != nil {
		return nil, errors.Wrapf(err, "write upstream")
	}

	switch applicationType {

	case "helm":
		defaultRelease := r.DefaultHelmRelease(localPath)
		return r.resolveRelease(
			ctx,
			upstream,
			localPath,
			constants.HelmChartPath,
			&defaultRelease,
			applicationType,
			true,
		)

	case "k8s":
		defaultRelease := r.DefaultRawRelease(constants.KustomizeBasePath)
		return r.resolveRelease(
			ctx,
			upstream,
			localPath,
			constants.KustomizeBasePath,
			&defaultRelease,
			applicationType,
			false,
		)

	case "replicated.app":
		parsed, err := url.Parse(upstream)
		if err != nil {
			return nil, errors.Wrapf(err, "parse url %s", upstream)
		}
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)
		return r.AppResolver.ResolveAppRelease(ctx, selector)
	case "inline.replicated.app":
		return r.resolveInlineShipYAMLRelease(
			ctx,
			upstream,
			localPath,
			applicationType,
		)

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

	defer func() {
		if err := r.FS.RemoveAll(localPath); err != nil {
			level.Error(r.Logger).Log("event", "remove watch dir", "err", err)
		}
	}()

	// this switch block is kinda duped from above, and we ought to centralize parts of this,
	// but in this case we only want to read the metadata without persisting anything to state,
	// and there doesn't seem to be a good way to evolve that abstraction cleanly from what we have, at least not just yet
	switch appType {
	case "helm":
		fallthrough
	case "k8s":
		fallthrough
	case "inline.replicated.app":
		metadata, err := r.ResolveBaseMetadata(upstream, localPath)
		if err != nil {
			return "", errors.Wrapf(err, "resolve metadata and content sha for %s %s", appType, upstream)
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

	return "", errors.Errorf("Could not continue with application type %q of upstream %s", appType, upstream)
}

func (r *Resolver) resolveRelease(
	ctx context.Context,
	upstream,
	localPath string,
	destPath string,
	defaultSpec *api.Spec,
	applicationType string,
	keepOriginal bool,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveChart")

	if r.Viper.GetBool("rm-asset-dest") {
		err := r.FS.RemoveAll(destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destPath)
		}
	}

	err := util.BailIfPresent(r.FS, destPath, debug)
	if err != nil {
		return nil, errors.Wrapf(err, "backup %s", destPath)
	}

	if !keepOriginal {
		err = r.FS.Rename(localPath, destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "move %s to %s", localPath, destPath)
		}
	} else {
		// instead of renaming, copy files from localPath to destPath
		err = r.recursiveCopy(localPath, destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "copy %s to %s", localPath, destPath)
		}
	}

	metadata, err := r.resolveMetadata(context.Background(), upstream, destPath, applicationType)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", destPath)
	}

	debug.Log("event", "check upstream for ship.yaml")
	spec, err := r.maybeGetShipYAML(ctx, destPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", destPath)
	}

	if spec == nil {
		debug.Log("event", "no ship.yaml for release")
		r.ui.Info("ship.yaml not found in upstream, generating default lifecycle for application ...")
		spec = defaultSpec
	}

	if applicationType == "k8s" {
		if err := r.maybeSplitMultidocYaml(ctx, destPath); err != nil {
			return nil, errors.Wrap(err, "split multipath yaml")
		}
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
		},
		Spec: *spec,
	}

	releaseName := release.Metadata.ReleaseName()
	debug.Log("event", "resolve.releaseName")

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	return release, nil
}

func (r *Resolver) recursiveCopy(sourceDir, destDir string) error {
	err := r.FS.MkdirAll(destDir, os.FileMode(0777))
	if err != nil {
		return errors.Wrapf(err, "create dest dir %s", destDir)
	}
	srcFiles, err := r.FS.ReadDir(sourceDir)
	if err != nil {
		return errors.Wrapf(err, "")
	}
	for _, file := range srcFiles {
		if file.IsDir() {
			err = r.recursiveCopy(filepath.Join(sourceDir, file.Name()), filepath.Join(destDir, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "copy dir %s", file.Name())
			}
		} else {
			// is file
			contents, err := r.FS.ReadFile(filepath.Join(sourceDir, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "read file %s to copy", file.Name())
			}

			err = r.FS.WriteFile(filepath.Join(destDir, file.Name()), contents, file.Mode())
			if err != nil {
				return errors.Wrapf(err, "write file %s to copy", file.Name())
			}
		}
	}
	return nil
}

func (r *Resolver) resolveInlineShipYAMLRelease(
	ctx context.Context,
	upstream,
	localPath string,
	applicationType string,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveInlineShipYAMLRelease")

	metadata, err := r.resolveMetadata(context.Background(), upstream, localPath, applicationType)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", localPath)
	}

	debug.Log("event", "check upstream for ship.yaml")
	spec, err := r.maybeGetShipYAML(ctx, localPath)
	if err != nil || spec == nil {
		return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", localPath)
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
		},
		Spec: *spec,
	}

	releaseName := release.Metadata.ReleaseName()
	debug.Log("event", "resolve.releaseName")

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	return release, nil
}
