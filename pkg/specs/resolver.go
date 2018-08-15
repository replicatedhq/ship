package specs

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helpers/flags"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

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
	Logger               log.Logger
	Client               *GraphQLClient
	GithubClient         *GithubClient
	StateManager         state.Manager
	FS                   afero.Afero
	Runbook              string
	SetChannelName       string
	RunbookReleaseSemver string
	SetChannelIcon       string
	HelmChartGitPath     string
	ui                   cli.Ui
}

// NewResolver builds a resolver from a Viper instance
func NewResolver(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	graphql *GraphQLClient,
	githubClient *GithubClient,
	stateManager state.Manager,
	ui cli.Ui,
) *Resolver {
	return &Resolver{
		Logger:               logger,
		Client:               graphql,
		GithubClient:         githubClient,
		StateManager:         stateManager,
		FS:                   fs,
		Runbook:              flags.GetCurrentOrDeprecatedString(v, "runbook", "studio-file"),
		SetChannelName:       flags.GetCurrentOrDeprecatedString(v, "set-channel-name", "studio-channel-name"),
		SetChannelIcon:       flags.GetCurrentOrDeprecatedString(v, "set-channel-icon", "studio-channel-icon"),
		RunbookReleaseSemver: v.GetString("release-semver"),
		HelmChartGitPath:     v.GetString("chart"),
		ui:                   ui,
	}
}

// ResolveRelease uses the passed config options to get specs from pg.replicated.com or
// from a local runbook if so configured
func (r *Resolver) ResolveRelease(ctx context.Context, selector Selector) (*api.Release, error) {
	var specYAML []byte
	var err error
	var release *ShipRelease

	debug := level.Debug(log.With(r.Logger, "method", "ResolveRelease"))

	if r.Runbook != "" {
		release, err = r.resolveRunbookRelease()
		if err != nil {
			return nil, errors.Wrapf(err, "resolve runbook from %s", r.Runbook)
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

	if err := yaml.Unmarshal([]byte(release.Spec), &result.Spec); err != nil {
		return nil, errors.Wrapf(err, "decode spec")
	}

	debug.Log("phase", "load-specs", "status", "complete",
		"resolved-spec", fmt.Sprintf("%+v", result.Spec),
	)

	return result, nil
}

func (r *Resolver) resolveRunbookRelease() (*ShipRelease, error) {
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
	debug.Log("phase", "write-yaml", "from", release.Spec, "write-location", constants.ReleasePath)

	return release, err
}

// persistSpec persists last-used YAML to disk at .ship/release.yml
func (r *Resolver) persistSpec(specYAML []byte) error {
	if err := r.FS.MkdirAll(filepath.Dir(constants.ReleasePath), 0700); err != nil {
		return errors.Wrap(err, "mkdir yaml")
	}

	if err := r.FS.WriteFile(constants.ReleasePath, specYAML, 0644); err != nil {
		return errors.Wrap(err, "write yaml file")
	}
	return nil
}

func (r *Resolver) RegisterInstall(ctx context.Context, selector Selector, release *api.Release) error {
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
