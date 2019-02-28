package replicatedapp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helpers/flags"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/state"
)

type shaSummer func([]byte) string
type resolver struct {
	Logger               log.Logger
	Client               *GraphQLClient
	FS                   afero.Afero
	StateManager         state.Manager
	UI                   cli.Ui
	ShaSummer            shaSummer
	Runbook              string
	SetChannelName       string
	RunbookReleaseSemver string
	SetChannelIcon       string
	SetGitHubContents    []string
	SetEntitlementsJSON  string
}

// NewAppResolver builds a resolver from a Viper instance
func NewAppResolver(
	v *viper.Viper,
	logger log.Logger,
	fs afero.Afero,
	graphql *GraphQLClient,
	stateManager state.Manager,
	ui cli.Ui,
) Resolver {
	return &resolver{
		Logger:               logger,
		Client:               graphql,
		FS:                   fs,
		UI:                   ui,
		Runbook:              flags.GetCurrentOrDeprecatedString(v, "runbook", "studio-file"),
		SetChannelName:       flags.GetCurrentOrDeprecatedString(v, "set-channel-name", "studio-channel-name"),
		SetChannelIcon:       flags.GetCurrentOrDeprecatedString(v, "set-channel-icon", "studio-channel-icon"),
		SetGitHubContents:    v.GetStringSlice("set-github-contents"),
		SetEntitlementsJSON:  v.GetString("set-entitlements-json"),
		RunbookReleaseSemver: v.GetString("release-semver"),
		StateManager:         stateManager,
		ShaSummer: func(bytes []byte) string {
			return fmt.Sprintf("%x", sha256.Sum256(bytes))
		},
	}
}

type Resolver interface {
	ResolveAppRelease(
		ctx context.Context,
		selector *Selector,
		app apptype.LocalAppCopy,
	) (*api.Release, error)
	FetchRelease(
		ctx context.Context,
		selector *Selector,
	) (*ShipRelease, error)
	RegisterInstall(
		ctx context.Context,
		selector Selector,
		release *api.Release,
	) error
	SetRunbook(
		runbook string,
	)
}

// ResolveAppRelease uses the passed config options to get specs from pg.replicated.com or
// from a local runbook if so configured
func (r *resolver) ResolveAppRelease(ctx context.Context, selector *Selector, app apptype.LocalAppCopy) (*api.Release, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveAppRelease"))
	release, err := r.FetchRelease(ctx, selector)
	if err != nil {
		return nil, errors.Wrap(err, "fetch release")
	}

	releaseName := release.ToReleaseMeta().ReleaseName()
	debug.Log("event", "resolve.releaseName")

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	result, err := r.persistRelease(release, selector)
	if err != nil {
		return nil, errors.Wrap(err, "persist and deserialize release")
	}

	result.Metadata.Type = app.GetType()

	return result, nil
}

// FetchRelease gets the release without persisting anything
func (r *resolver) FetchRelease(ctx context.Context, selector *Selector) (*ShipRelease, error) {
	var specYAML []byte
	var err error
	var release *ShipRelease

	debug := level.Debug(log.With(r.Logger, "method", "FetchRelease"))
	if r.Runbook != "" {
		release, err = r.resolveRunbookRelease(selector)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve runbook from %s", r.Runbook)
		}
	} else {
		release, err = r.resolveCloudRelease(selector)
		debug.Log("event", "spec.resolve", "spec", specYAML, "err", err)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve gql spec for %s", selector)
		}
	}
	debug.Log("event", "spec.resolve.success", "spec", specYAML, "err", err)
	return release, nil
}

func (r *resolver) persistRelease(release *ShipRelease, selector *Selector) (*api.Release, error) {
	debug := level.Debug(log.With(r.Logger, "method", "persistRelease"))

	result := &api.Release{
		Metadata: release.ToReleaseMeta(),
	}
	result.Metadata.CustomerID = selector.CustomerID
	result.Metadata.InstallationID = selector.InstallationID
	result.Metadata.LicenseID = selector.LicenseID
	result.Metadata.AppSlug = selector.AppSlug

	if err := r.StateManager.SerializeAppMetadata(result.Metadata); err != nil {
		return nil, errors.Wrap(err, "serialize app metadata")
	}

	contentSHA := r.ShaSummer([]byte(release.Spec))
	if err := r.StateManager.SerializeContentSHA(contentSHA); err != nil {
		return nil, errors.Wrap(err, "serialize content sha")
	}

	if err := yaml.Unmarshal([]byte(release.Spec), &result.Spec); err != nil {
		return nil, errors.Wrapf(err, "decode spec")
	}
	debug.Log("phase", "load-specs", "status", "complete",
		"resolved-spec", fmt.Sprintf("%+v", result.Spec),
	)
	return result, nil
}

func (r *resolver) resolveCloudRelease(selector *Selector) (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveCloudSpec"))

	var release *ShipRelease
	var err error
	client := r.Client
	if selector.CustomerID != "" {
		debug.Log("phase", "load-specs", "from", "gql", "addr", client.GQLServer.String(), "method", "customerID")
		release, err = client.GetRelease(selector)
		if err != nil {
			return nil, err
		}
	} else {
		debug.Log("phase", "load-specs", "from", "gql", "addr", client.GQLServer.String(), "method", "appSlug")
		if selector.AppSlug == "" {
			return nil, errors.New("either a customer ID or app slug must be provided")
		}
		release, err = client.GetSlugRelease(selector)
		if err != nil {
			if selector.LicenseID == "" {
				debug.Log("event", "spec-resolve", "from", selector, "error", err)

				var input string
				input, err = r.UI.Ask("Please enter your license to continue: ")
				if err != nil {
					return nil, errors.Wrapf(err, "enter license from CLI")
				}

				selector.LicenseID = input

				err = r.updateUpstream(*selector)
				if err != nil {
					return nil, errors.Wrapf(err, "persist updated upstream")
				}

				release, err = client.GetSlugRelease(selector)
			}

			if err != nil {
				return nil, err
			}
		}
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

func (r *resolver) SetRunbook(runbook string) {
	r.Runbook = runbook
}

func (r *resolver) loadFakeEntitlements() (*api.Entitlements, error) {
	var entitlements api.Entitlements
	err := json.Unmarshal([]byte(r.SetEntitlementsJSON), &entitlements)
	if err != nil {
		return nil, errors.Wrap(err, "load entitlements json")
	}
	return &entitlements, nil
}

// read the upstream, get the host/path, and replace the query params with the ones from the provided selector
func (r *resolver) updateUpstream(selector Selector) error {
	currentState, err := r.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "retrieve state")
	}
	currentUpstream := currentState.Upstream()

	parsedUpstream, err := url.Parse(currentUpstream)
	if err != nil {
		return errors.Wrap(err, "parse upstream")
	}

	if !strings.HasSuffix(parsedUpstream.Path, "/") {
		parsedUpstream.Path += "/"
	}

	return r.StateManager.SerializeUpstream(parsedUpstream.Path + "?" + selector.String())
}
