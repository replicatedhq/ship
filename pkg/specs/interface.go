package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/util"
)

func (r *Resolver) ResolveUnforkRelease(ctx context.Context, upstream string, forked string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveUnforkReleases")
	r.ui.Info(fmt.Sprintf("Reading %s and %s ...", upstream, forked))

	// Prepare the upstream
	r.ui.Info("Determining upstream application type ...")
	upstreamApplicationType, localUpstreamPath, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", upstream)
	}
	debug.Log("event", "applicationType.resolve", "type", upstreamApplicationType)
	r.ui.Info(fmt.Sprintf("Detected upstream application type %s", upstreamApplicationType))

	debug.Log("event", "versionedUpstream.resolve", "type", upstreamApplicationType)
	versionedUpstream, err := r.maybeCreateVersionedUpstream(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "resolve versioned upstream")
	}

	debug.Log("event", "upstream.Serialize", "for", localUpstreamPath, "upstream", versionedUpstream)
	err = r.StateManager.SerializeUpstream(versionedUpstream)
	if err != nil {
		return nil, errors.Wrapf(err, "write upstream")
	}

	// Prepare the fork
	r.ui.Info("Determining forked application type ...")
	forkedApplicationType, localForkedPath, err := r.appTypeInspector.DetermineApplicationType(ctx, forked)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", forked)
	}

	debug.Log("event", "applicationType.resolve", "type", forkedApplicationType)
	r.ui.Info(fmt.Sprintf("Detected forked application type %s", forkedApplicationType))

	if forkedApplicationType == "helm" && upstreamApplicationType == "k8s" {
		return nil, errors.New("Unsupported fork and upstream combination")
	}

	forkedAsset := api.Asset{}
	switch forkedApplicationType {
	case "helm":
		forkedAsset = api.Asset{
			Helm: &api.HelmAsset{
				AssetShared: api.AssetShared{
					Dest: constants.UnforkForkedBasePath,
				},
				Local: &api.LocalHelmOpts{
					ChartRoot: constants.HelmChartForkedPath,
				},
				ValuesFrom: &api.ValuesFrom{
					Path:        filepath.Join(constants.HelmChartForkedPath),
					SaveToState: true,
				},
			},
		}
	case "k8s":
		forkedAsset = api.Asset{
			Local: &api.LocalAsset{
				AssetShared: api.AssetShared{
					Dest: constants.UnforkForkedBasePath,
				},
				Path: constants.HelmChartForkedPath,
			},
		}
	default:
		return nil, errors.Errorf("unknown forked application type %q", forkedApplicationType)
	}

	upstreamAsset := api.Asset{}
	switch upstreamApplicationType {
	case "helm":
		upstreamAsset = api.Asset{
			Helm: &api.HelmAsset{
				AssetShared: api.AssetShared{
					Dest: constants.KustomizeBasePath,
				},
				Local: &api.LocalHelmOpts{
					ChartRoot: constants.HelmChartPath,
				},
				ValuesFrom: &api.ValuesFrom{
					Lifecycle: &api.ValuesFromLifecycle{},
				},
			},
		}
	case "k8s":
		upstreamAsset = api.Asset{
			Local: &api.LocalAsset{
				AssetShared: api.AssetShared{
					Dest: constants.KustomizeBasePath,
				},
				Path: constants.HelmChartPath,
			},
		}
	default:
		return nil, errors.Errorf("unknown upstream application type %q", forkedApplicationType)
	}

	defaultRelease := r.DefaultHelmUnforkRelease(upstreamAsset, forkedAsset)

	return r.resolveUnforkRelease(
		ctx,
		upstream,
		forked,
		localUpstreamPath,
		localForkedPath,
		constants.HelmChartPath,
		constants.HelmChartForkedPath,
		&defaultRelease,
		upstreamApplicationType,
		forkedApplicationType,
	)
}

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
			true,
		)

	case "runbook.replicated.app":
		r.AppResolver.SetRunbook(localPath)
		fallthrough
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

	case "runbook.replicated.app":
		r.AppResolver.SetRunbook(localPath)
		fallthrough
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

func (r *Resolver) resolveUnforkRelease(
	ctx context.Context,
	upstream string,
	forked string,
	localUpstreamPath string,
	localForkedPath string,
	destUpstreamPath string,
	destForkedPath string,
	defaultSpec *api.Spec,
	applicationType string,
	forkedApplicationType string,
) (*api.Release, error) {
	var releaseName string
	debug := log.With(level.Debug(r.Logger), "method", "resolveUnforkReleases")

	if r.Viper.GetBool("rm-asset-dest") {
		err := r.FS.RemoveAll(destUpstreamPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destUpstreamPath)
		}

		err = r.FS.RemoveAll(destForkedPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destForkedPath)
		}
	}

	err := util.BailIfPresent(r.FS, destUpstreamPath, debug)
	if err != nil {
		return nil, errors.Wrapf(err, "backup %s", destUpstreamPath)
	}

	err = r.FS.MkdirAll(filepath.Dir(destUpstreamPath), 0777)
	if err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", localUpstreamPath)
	}

	err = r.FS.MkdirAll(filepath.Dir(destForkedPath), 0777)
	if err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", destForkedPath)
	}

	err = r.FS.Rename(localUpstreamPath, destUpstreamPath)
	if err != nil {
		return nil, errors.Wrapf(err, "move %s to %s", localUpstreamPath, destUpstreamPath)
	}

	err = r.FS.Rename(localForkedPath, destForkedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "move %s to %s", localForkedPath, destForkedPath)
	}

	if forkedApplicationType == "k8s" {
		// Pre-emptively need to split here in order to get the release name before
		// helm template is run on the upstream
		if err := util.MaybeSplitMultidocYaml(ctx, r.FS, destForkedPath); err != nil {
			return nil, errors.Wrapf(err, "maybe split multidoc in %s", destForkedPath)
		}

		debug.Log("event", "maybeGetReleaseName")
		releaseName, err = r.maybeGetReleaseName(destForkedPath)
		if err != nil {
			return nil, errors.Wrap(err, "maybe get release name")
		}
	}

	upstreamMetadata, err := r.resolveMetadata(context.Background(), upstream, destUpstreamPath, applicationType)
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", destUpstreamPath)
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *upstreamMetadata,
		},
		Spec: *defaultSpec,
	}

	if releaseName == "" {
		releaseName = release.Metadata.ReleaseName()
	}

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	return release, nil
}

func (r *Resolver) maybeGetReleaseName(path string) (string, error) {
	type k8sReleaseMetadata struct {
		Metadata struct {
			Labels struct {
				Release string `yaml:"release"`
			} `yaml:"labels"`
		} `yaml:"metadata"`
	}

	files, err := r.FS.ReadDir(path)
	if err != nil {
		return "", errors.Wrapf(err, "read dir %s", path)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
			fileB, err := r.FS.ReadFile(filepath.Join(path, file.Name()))
			if err != nil {
				return "", errors.Wrapf(err, "read file %s", path)
			}

			releaseMetadata := k8sReleaseMetadata{}
			if err := yaml.Unmarshal(fileB, &releaseMetadata); err != nil {
				return "", errors.Wrapf(err, "unmarshal for release metadata %s", path)
			}

			if releaseMetadata.Metadata.Labels.Release != "" {
				return releaseMetadata.Metadata.Labels.Release, nil
			}
		}
	}

	return "", nil
}

func (r *Resolver) resolveRelease(
	ctx context.Context,
	upstream,
	localPath string,
	destPath string,
	defaultSpec *api.Spec,
	applicationType string,
	keepOriginal bool,
	tryUseUpstreamShipYAML bool,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveRelease")

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

	var spec *api.Spec
	if tryUseUpstreamShipYAML {
		debug.Log("event", "check upstream for ship.yaml")
		spec, err = r.maybeGetShipYAML(ctx, destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", destPath)
		}
	}

	if spec == nil {
		debug.Log("event", "no ship.yaml for release")
		r.ui.Info("ship.yaml not found in upstream, generating default lifecycle for application ...")
		spec = defaultSpec
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
		},
		Spec: *spec,
	}

	currentState, err := r.StateManager.TryLoad()
	if err != nil {
		return nil, errors.Wrap(err, "try load")
	}

	releaseName := currentState.CurrentReleaseName()
	if releaseName == "" {
		debug.Log("event", "resolve.releaseName.fromRelease")
		releaseName = release.Metadata.ReleaseName()
	}

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
