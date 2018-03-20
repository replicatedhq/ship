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
	AllowInlineSpecs = true // We might set this to false in the prod build, IDK
)

type Resolver struct {
	Logger     log.Logger
	StudioFile string
}

func ResolverFromViper(v *viper.Viper) *Resolver {
	return &Resolver{
		Logger:     logger.FromViper(v),
		StudioFile: v.GetString("studio_file"),
	}
}

func (r *Resolver) ResolveSpecs(ctx context.Context) (*api.Spec, error) {
	var spec api.Spec
	var rawSpec map[string]interface{}
	var jsonSpecForDebug []byte

	debug := level.Debug(log.With(r.Logger, "method", "configure"))

	if r.StudioFile != "" && AllowInlineSpecs {
		debug.Log("phase", "load-specs", "from", "studio_file", "file", r.StudioFile)
		specHCL, err := ioutil.ReadFile(r.StudioFile)
		if err != nil {
			return nil, errors.Wrapf(err, "read specs from %s", r.StudioFile)
		}
		debug.Log("phase", "load-specs", "from", "studio_file", "file", r.StudioFile, "spec", specHCL)

		if err := yaml.Unmarshal(specHCL, &spec); err != nil {
			return nil, errors.Wrapf(err, "decode specs from %s", r.StudioFile)
		}
	}

	// else load specs from GraphQL

	debug.Log("phase", "load-specs",
		"resolved-spec", fmt.Sprintf("%+v", spec),
		"resolved-spec-raw", fmt.Sprintf("%+v", rawSpec),
		"resolved-spec-raw-json", fmt.Sprintf("%s", jsonSpecForDebug),
	)

	return &spec, nil
}
