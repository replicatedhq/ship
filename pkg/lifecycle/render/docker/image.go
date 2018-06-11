package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/api"
)

type PullURLResolver interface {
	ResolvePullURL(asset *api.DockerAsset, meta api.ReleaseMetadata) (string, error)
}

var _ PullURLResolver = &URLResolver{}

type URLResolver struct {
	Logger log.Logger
}

func URLResolverFromViper(logger log.Logger) PullURLResolver {
	return &URLResolver{
		Logger: logger,
	}
}

func (r *URLResolver) ResolvePullURL(asset *api.DockerAsset, meta api.ReleaseMetadata) (string, error) {
	debug := level.Debug(r.Logger)

	if asset.Source == "replicated" || asset.Source == "public" || asset.Source == "" {
		debug.Log("event", "image.rewrite.skip", "source", asset.Source)
		return asset.Image, nil
	}

	for _, image := range meta.Images {
		if image.URL != asset.Image {
			continue
		}

		imageName, imageTag, err := resolveImageName(asset.Image)
		if err != nil {
			return "", errors.Wrapf(err, "parse image url %s", asset.Image)
		}

		url := fmt.Sprintf("%s/%s/%s.%s:%s", replicatedRegistry(), image.AppSlug, image.ImageKey, imageName, imageTag)
		return url, nil
	}

	return asset.Image, nil
}

func replicatedRegistry() string {
	reg := os.Getenv("REPLICATED_REGISTRY")
	if reg != "" {
		return reg
	}
	return "registry.replicated.com"
}

func resolveImageName(url string) (string, string, error) {
	ref, err := reference.ParseNormalizedNamed(url)
	if err != nil {
		return "", "", err
	}

	var name, tag string
	switch x := ref.(type) {
	case reference.NamedTagged:
		name = reference.Path(x)
		tag = x.Tag()
	default:
		name = reference.Path(x)
		tag = "latest"
	}

	parts := strings.Split(name, "/")
	if len(parts) == 1 {
		return parts[0], tag, nil
	}
	if len(parts) == 2 {
		return parts[1], tag, nil
	}

	return "", "", fmt.Errorf("unsupported image path %s", name)
}
