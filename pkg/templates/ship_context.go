package templates

import (
	"text/template"

	"github.com/go-kit/kit/log"
)

var amazonEKSPaths map[string]string
var googleGKEPaths map[string]string

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
		"AmazonEKS": ctx.amazonEKS,
		"GoogleGKE": ctx.googleGKE,
	}
}

// amazonEKS returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) amazonEKS(name string) string {
	return amazonEKSPaths[name]
}

// AddAmazonEKSPath adds a kubeconfig path to the cache
func AddAmazonEKSPath(name string, path string) {
	if amazonEKSPaths == nil {
		amazonEKSPaths = make(map[string]string)
	}
	amazonEKSPaths[name] = path
}

// googleGKE returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) googleGKE(name string) string {
	return googleGKEPaths[name]
}

// AddGoogleGKEPath adds a kubeconfig path to the cache
func AddGoogleGKEPath(name string, path string) {
	if googleGKEPaths == nil {
		googleGKEPaths = make(map[string]string)
	}
	googleGKEPaths[name] = path
}
