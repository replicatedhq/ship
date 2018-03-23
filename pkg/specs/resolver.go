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

// ResolveSpecs uses the passed config options to get specs from pg.replicated.com or
// from a local studio-file if so configured
func (r *Resolver) ResolveSpecs(ctx context.Context, customerID string) (*api.Spec, error) {
	var specYAML []byte
	var err error
	var spec api.Spec
	var rawSpec map[string]interface{}
	var jsonSpecForDebug []byte

	debug := level.Debug(log.With(r.Logger, "method", "ResolveSpecs"))

	if r.StudioFile != "" && AllowInlineSpecs {
		specYAML, err = r.resolveStudioSpec()
		if err != nil {
			return nil, errors.Wrapf(err, "resolve studio spec from %s", r.StudioFile)
		}
	} else {
		specYAML, err = r.resolveCloudSpec(customerID)
		debug.Log("spec.resolve", "spec", specYAML, "err", err)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve gql spec for %s", customerID)
		}
	}

	if err := yaml.Unmarshal(specYAML, &spec); err != nil {
		return nil, errors.Wrapf(err, "decode spec", r.StudioFile)

	}

	debug.Log("phase", "load-specs",
		"resolved-spec", fmt.Sprintf("%+v", spec),
		"resolved-spec-raw", fmt.Sprintf("%+v", rawSpec),
		"resolved-spec-raw-json", fmt.Sprintf("%s", jsonSpecForDebug),
	)

	return &spec, nil
}

func (r *Resolver) resolveStudioSpec() ([]byte, error) {

	debug := level.Debug(log.With(r.Logger, "method", "resolveStudioSpec"))
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile)
	specYAML, err := ioutil.ReadFile(r.StudioFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read specs from %s", r.StudioFile)
	}
	debug.Log("phase", "load-specs", "from", "studio-file", "file", r.StudioFile, "spec", specYAML)
	return specYAML, nil
}

func (r *Resolver) resolveCloudSpec(customerID string) ([]byte, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveCloudSpec"))

	debug.Log("phase", "load-specs", "from", "gql", "addr", r.Client.GQLServer.String())
	spec, err := r.Client.GetRelease(customerID, "")
	return []byte(spec.Spec), err

}
