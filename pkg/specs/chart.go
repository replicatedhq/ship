package specs

import (
	"context"
	"io/ioutil"

	"github.com/replicatedhq/ship/pkg/api"
	"gopkg.in/yaml.v2"
)

func resolveChartMetadata(ctx context.Context, path string) (api.HelmChartMetadata, error) {
	var md api.HelmChartMetadata

	chart, err := ioutil.ReadFile(path)
	if err != nil {
		return api.HelmChartMetadata{}, err
	}

	if err := yaml.Unmarshal(chart, &md); err != nil {
		return api.HelmChartMetadata{}, err
	}

	return md, nil
}
