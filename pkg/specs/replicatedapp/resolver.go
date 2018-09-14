package replicatedapp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helpers/flags"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type resolver struct {
	Logger               log.Logger
	Client               *GraphQLClient
	FS                   afero.Afero
	StateManager         state.Manager
	Runbook              string
	SetChannelName       string
	RunbookReleaseSemver string
	SetChannelIcon       string
}

// NewAppResolver builds a resolver from a Viper instance
func NewAppResolver(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	graphql *GraphQLClient,
	stateManager state.Manager,
) Resolver {
	return &resolver{
		Logger:               logger,
		Client:               graphql,
		FS:                   fs,
		Runbook:              flags.GetCurrentOrDeprecatedString(v, "runbook", "studio-file"),
		SetChannelName:       flags.GetCurrentOrDeprecatedString(v, "set-channel-name", "studio-channel-name"),
		SetChannelIcon:       flags.GetCurrentOrDeprecatedString(v, "set-channel-icon", "studio-channel-icon"),
		RunbookReleaseSemver: v.GetString("release-semver"),
		StateManager:         stateManager,
	}
}

type Resolver interface {
	ResolveAppRelease(
		ctx context.Context,
		selector *Selector,
	) (*api.Release, error)
	RegisterInstall(
		ctx context.Context,
		selector Selector,
		release *api.Release,
	) error
}

// ResolveAppRelease uses the passed config options to get specs from pg.replicated.com or
// from a local runbook if so configured
func (r *resolver) ResolveAppRelease(ctx context.Context, selector *Selector) (*api.Release, error) {
	var specYAML []byte
	var err error
	var release *ShipRelease

	debug := level.Debug(log.With(r.Logger, "method", "ResolveAppRelease"))

	if r.Runbook != "" {
		release, err = r.resolveRunbookRelease()
		if err != nil {
			return nil, errors.Wrapf(err, "resolve runbook from %s", r.Runbook)
		}
	} else {
		release, err = r.resolveCloudRelease(selector)
		debug.Log("spec.resolve", "spec", specYAML, "err", err)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve gql spec for %s", selector)
		}
	}

	result := &api.Release{
		Metadata: release.ToReleaseMeta(),
	}
	result.Metadata.CustomerID = selector.CustomerID
	result.Metadata.InstallationID = selector.InstallationID

	if err := r.StateManager.SerializeAppMetadata(result.Metadata); err != nil {
		return nil, errors.Wrap(err, "serialize app metadata")
	}

	if err := yaml.Unmarshal([]byte(release.Spec), &result.Spec); err != nil {
		return nil, errors.Wrapf(err, "decode spec")
	}

	debug.Log("phase", "load-specs", "status", "complete",
		"resolved-spec", fmt.Sprintf("%+v", result.Spec),
	)

	return result, nil
}

func (r *resolver) resolveRunbookRelease() (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveRunbookRelease"))
	debug.Log("phase", "load-specs", "from", "runbook", "file", r.Runbook)

	specYAML, err := r.FS.ReadFile(r.Runbook)
	if err != nil {
		return nil, errors.Wrapf(err, "read specs from %s", r.Runbook)
	}
	debug.Log("phase", "load-specs", "from", "runbook", "file", r.Runbook, "spec", specYAML)

	if err := r.persistSpec(specYAML); err != nil {
		return nil, errors.Wrapf(err, "serialize last-used YAML to disk")
	}
	debug.Log("phase", "write-yaml", "from", r.Runbook, "write-location", constants.ReleasePath)

	return &ShipRelease{
		Spec:        string(specYAML),
		ChannelName: r.SetChannelName,
		ChannelIcon: r.SetChannelIcon,
		Semver:      r.RunbookReleaseSemver,
	}, nil
}

func (r *resolver) resolveCloudRelease(selector *Selector) (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveCloudSpec"))

	client := r.Client
	debug.Log("phase", "load-specs", "from", "gql", "addr", client.GQLServer.String())
	release, err := client.GetRelease(selector)
	if err != nil {
		return nil, err
	}

	if err := r.persistSpec([]byte(release.Spec)); err != nil {
		return nil, errors.Wrapf(err, "serialize last-used YAML to disk")
	}
	debug.Log("phase", "write-yaml", "from", release.Spec, "write-location", constants.ReleasePath)

	return release, err
}

// persistSpec persists last-used YAML to disk at .ship/release.yml
func (r *resolver) persistSpec(specYAML []byte) error {
	if err := r.FS.MkdirAll(filepath.Dir(constants.ReleasePath), 0700); err != nil {
		return errors.Wrap(err, "mkdir yaml")
	}

	if err := r.FS.WriteFile(constants.ReleasePath, specYAML, 0644); err != nil {
		return errors.Wrap(err, "write yaml file")
	}
	return nil
}

func (r *resolver) RegisterInstall(ctx context.Context, selector Selector, release *api.Release) error {
	if r.Runbook != "" {
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
