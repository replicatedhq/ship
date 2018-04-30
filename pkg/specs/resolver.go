package specs

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	// AllowInlineSpecs enables the use of a local file instead of a properly-licensed customer ID
	// we might set this to false in the prod build, or just refuse to manage state if studio is used, not sure
	AllowInlineSpecs = true
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
	Logger     log.Logger
	StudioFile string
	Client     *GraphQLClient
}

// ResolverFromViper builds a resolver from a Viper instance
func ResolverFromViper(v *viper.Viper) (*Resolver, error) {
	graphql, err := GraphQLClientFromViper(v)
	if err != nil {
		return nil, errors.Wrap(err, "get graphql client")
	}
	return &Resolver{
		Logger:     logger.FromViper(v),
		StudioFile: v.GetString("studio-file"),
		Client:     graphql,
	}, nil
}

// ResolveRelease uses the passed config options to get specs from pg.replicated.com or
// from a local studio-file if so configured
func (r *Resolver) ResolveRelease(ctx context.Context, selector Selector) (*api.Release, error) {
	var specYAML []byte
	var err error
	var release *ShipRelease

	debug := level.Debug(log.With(r.Logger, "method", "ResolveSpecs"))

	if r.StudioFile != "" && AllowInlineSpecs {
		release, err = r.resolveStudioRelease()
		if err != nil {
			return nil, errors.Wrapf(err, "resolve studio spec from %s", r.StudioFile)
		}
	} else {
		release, err = r.resolveCloudRelease(selector.CustomerID)
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

func (r *Resolver) resolveStudioRelease() (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveStudioSpec"))
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile)

	specYAML, err := ioutil.ReadFile(r.StudioFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read specs from %s", r.StudioFile)
	}
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile, "spec", specYAML)
	return &ShipRelease{Spec: string(specYAML)}, nil
}

func (r *Resolver) resolveCloudRelease(customerID string) (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveCloudSpec"))

	client := r.Client
	debug.Log("phase", "load-specs", "from", "gql", "addr", client.GQLServer.String())
	release, err := client.GetRelease(customerID, "")
	if err != nil {
		return nil, err
	}
	return release, err
}
