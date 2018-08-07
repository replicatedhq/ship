package templates

import (
	"text/template"

	"github.com/go-kit/kit/log"
)

var AmazonElasticKubernetesServicePath map[string]string

// ShipContext is the context for builder functions that depend on what assets have been created.
type ShipContext struct {
	Logger log.Logger
}

func (bb *BuilderBuilder) NewShipContext() (*ShipContext, error) {
	shipCtx := &ShipContext{
		Logger: bb.Logger,
	}

	return shipCtx, nil
}

// FuncMap represents the available functions in the ConfigCtx.
func (ctx ShipContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"AmazonElasticKubernetesService": ctx.amazonElasticKubernetesService,
	}
}

// amazonElasticKubernetesService returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) amazonElasticKubernetesService(name string) string {
	return AmazonElasticKubernetesServicePath[name]
}

// AddAmazonElasticKubernetesServicePath adds a kubeconfig path to the cache
func AddAmazonElasticKubernetesServicePath(name string, path string) {
	if AmazonElasticKubernetesServicePath == nil {
		AmazonElasticKubernetesServicePath = make(map[string]string)
	}
	AmazonElasticKubernetesServicePath[name] = path
}
