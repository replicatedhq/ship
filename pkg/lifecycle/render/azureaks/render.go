package azureaks

import (
	"bytes"
	"context"
	"encoding/base64"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-kit/kit/log"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset (that produces an AKS cluster) as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.AKSAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// a LocalRenderer renders a terraform asset by writing generated terraform source code
type LocalRenderer struct {
	Logger         log.Logger
	BuilderBuilder *templates.BuilderBuilder
	Inline         inline.Renderer
	Fs             afero.Afero
}

var _ Renderer = &LocalRenderer{}

func NewRenderer(
	logger log.Logger,
	bb *templates.BuilderBuilder,
	inline inline.Renderer,
	fs afero.Afero,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		BuilderBuilder: bb,
		Inline:         inline,
		Fs:             fs,
	}
}

func (r *LocalRenderer) Execute(
	rootFs root.Fs,
	asset api.AKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		builder, err := r.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "init builder")
		}

		asset, err = buildAsset(asset, builder)
		if err != nil {
			return errors.Wrap(err, "build asset")
		}

		assetsPath := "azure_aks.tf"
		if asset.Dest != "" {
			assetsPath = asset.Dest
		}
		kubeConfigPath := path.Join(path.Dir(assetsPath), "kubeconfig_"+asset.ClusterName)

		contents, err := renderTerraformContents(asset, kubeConfigPath)
		if err != nil {
			return errors.Wrap(err, "render tf config")
		}

		templates.AddAzureAKSPath(asset.ClusterName, kubeConfigPath)

		// write the inline spec
		err = r.Inline.Execute(
			rootFs,
			api.InlineAsset{
				Contents: contents,
				AssetShared: api.AssetShared{
					Dest: assetsPath,
					Mode: asset.Mode,
				},
			},
			meta,
			templateContext,
			configGroups,
		)(ctx)

		if err != nil {
			return errors.Wrap(err, "write tf config")
		}
		return nil
	}
}

// build asset values used outside of the terraform inline asset
func buildAsset(asset api.AKSAsset, builder *templates.Builder) (api.AKSAsset, error) {
	var err error
	var multiErr *multierror.Error

	asset.ClusterName, err = builder.String(asset.ClusterName)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build cluster_name"))

	asset.Dest, err = builder.String(asset.Dest)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build dest"))

	return asset, multiErr.ErrorOrNil()
}

func renderTerraformContents(asset api.AKSAsset, kubeConfigPath string) (string, error) {
	t, err := template.New("aksTemplate").
		Funcs(sprig.TxtFuncMap()).
		Parse(clusterTempl)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, asset, kubeConfigPath)
}

func executeTemplate(t *template.Template, asset api.AKSAsset, kubeConfigPath string) (string, error) {
	var data = struct {
		api.AKSAsset
		KubeConfigPath  string
		SafeClusterName string
	}{
		asset,
		kubeConfigPath,
		safeClusterName(asset.ClusterName),
	}
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

// Create a string from the clusterName safe for use as the agent pool name and
// the dns_prefix.
// "Agent Pool names must start with a lowercase letter, have max length of 12, and only have characters a-z0-9"
var unsafeClusterNameChars = regexp.MustCompile(`[^a-z0-9]`)
var startsWithLower = regexp.MustCompile(`^[a-z]`)

func safeClusterName(clusterName string) string {
	s := strings.ToLower(clusterName)
	s = unsafeClusterNameChars.ReplaceAllString(s, "")
	for !startsWithLower.MatchString(s) && len(s) > 0 {
		s = s[1:]
	}
	if len(s) > 12 {
		return s[0:12]
	}
	if len(s) == 0 {
		return safeClusterName(base64.StdEncoding.EncodeToString([]byte("cluster" + clusterName)))
	}
	return s
}
