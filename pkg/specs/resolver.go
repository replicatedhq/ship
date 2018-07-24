package specs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/state"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const ReleasePath = ".ship/release.yml"

// Selector selects a spec from the Vendor's releases and channels.
// See pkg/cli/root.go for some more info on which are required and why.
type Selector struct {
	// required
	CustomerID string

	// optional
	ReleaseSemver  string
	ReleaseID      string
	ChannelID      string
	InstallationID string
}

// A Resolver resolves specs
type Resolver struct {
	Logger              log.Logger
	Client              *GraphQLClient
	StateManager        *state.Manager
	StudioFile          string
	StudioChannelName   string
	StudioReleaseSemver string
	StudioChannelIcon   string
	HelmChartPath       string
}

// NewResolver builds a resolver from a Viper instance
func NewResolver(
	v *viper.Viper,
	logger log.Logger,
	graphql *GraphQLClient,
	stateManager *state.Manager,
) *Resolver {
	return &Resolver{
		Logger:              logger,
		Client:              graphql,
		StateManager:        stateManager,
		StudioFile:          v.GetString("studio-file"),
		StudioChannelName:   v.GetString("studio-channel-name"),
		StudioChannelIcon:   v.GetString("studio-channel-icon"),
		StudioReleaseSemver: v.GetString("release-semver"),
		HelmChartPath:       v.GetString("file"),
	}
}

// ResolveRelease uses the passed config options to get specs from pg.replicated.com or
// from a local studio-file if so configured
func (r *Resolver) ResolveRelease(ctx context.Context, selector Selector) (*api.Release, error) {
	var specYAML []byte
	var err error
	var release *ShipRelease

	debug := level.Debug(log.With(r.Logger, "method", "ResolveRelease"))

	if r.StudioFile != "" {
		release, err = r.resolveStudioRelease()
		if err != nil {
			return nil, errors.Wrapf(err, "resolve studio spec from %s", r.StudioFile)
		}
	} else {
		release, err = r.resolveCloudRelease(selector.CustomerID, selector.InstallationID, selector.ReleaseSemver)
		debug.Log("spec.resolve", "spec", specYAML, "err", err)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve gql spec for %s", selector.CustomerID)
		}
	}

	result := &api.Release{
		Metadata: release.ToReleaseMeta(),
	}
	result.Metadata.CustomerID = selector.CustomerID

	if r.HelmChartPath != "" {
		result.Metadata.HelmChartMetadata, err = resolveChartMetadata(ctx, r.HelmChartPath)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve helm metadata for %s", r.HelmChartPath)
		}
	}

	if err := yaml.Unmarshal([]byte(release.Spec), &result.Spec); err != nil {
		return nil, errors.Wrapf(err, "decode spec")
	}

	debug.Log("phase", "load-specs", "status", "complete",
		"resolved-spec", fmt.Sprintf("%+v", result.Spec),
	)

	return result, nil
}

func (r *Resolver) resolveStudioRelease() (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveStudioSpec"))
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile)

	specYAML, err := r.StateManager.FS.ReadFile(r.StudioFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read specs from %s", r.StudioFile)
	}
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile, "spec", specYAML)

	if err := r.persistSpec(specYAML); err != nil {
		return nil, errors.Wrapf(err, "serialize last-used YAML to disk")
	}
	debug.Log("phase", "write-yaml", "from", r.StudioFile, "write-location", ReleasePath)

	return &ShipRelease{
		Spec:        string(specYAML),
		ChannelName: r.StudioChannelName,
		ChannelIcon: r.StudioChannelIcon,
		Semver:      r.StudioReleaseSemver,
	}, nil
}

func (r *Resolver) resolveCloudRelease(customerID, installationID, semver string) (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveCloudSpec"))

	client := r.Client
	debug.Log("phase", "load-specs", "from", "gql", "addr", client.GQLServer.String())
	release, err := client.GetRelease(customerID, installationID, semver)
	if err != nil {
		return nil, err
	}

	if err := r.persistSpec([]byte(release.Spec)); err != nil {
		return nil, errors.Wrapf(err, "serialize last-used YAML to disk")
	}
	debug.Log("phase", "write-yaml", "from", release.Spec, "write-location", ReleasePath)

	return release, err
}

// persistSpec persists last-used YAML to disk at .ship/release.yml
func (r *Resolver) persistSpec(specYAML []byte) error {
	if err := r.StateManager.FS.MkdirAll(filepath.Dir(ReleasePath), 0700); err != nil {
		return errors.Wrap(err, "mkdir yaml")
	}

	if err := r.StateManager.FS.WriteFile(ReleasePath, specYAML, 0644); err != nil {
		return errors.Wrap(err, "write yaml file")
	}
	return nil
}

func (r *Resolver) RegisterInstall(ctx context.Context, selector Selector, release *api.Release) error {
	if r.StudioFile != "" {
		return nil
	}

	debug := level.Debug(log.With(r.Logger, "method", "RegisterRelease"))

	debug.Log("phase", "register", "with", "gql", "addr", r.Client.GQLServer.String())

	err := r.Client.RegisterInstall(selector.CustomerID, "", release.Metadata.ChannelID, release.Metadata.ReleaseID)
	if err != nil {
		return err
	}

	debug.Log("phase", "register", "status", "complete")

	return nil
}
