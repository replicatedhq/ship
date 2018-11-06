package googlegke

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset (that produces an GKE cluster) as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
		asset api.GKEAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// LocalRenderer renders a terraform asset by writing generated terraform source code
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
	asset api.GKEAsset,
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

		contents, err := renderTerraformContents(asset)
		if err != nil {
			return errors.Wrap(err, "render tf config")
		}
		fmt.Println("contents", contents)

		assetsPath := "google_gke.tf"
		if asset.Dest != "" {
			assetsPath = asset.Dest
		}

		// save the path to the kubeconfig that running the generated terraform will produce
		templates.AddGoogleGKEPath(asset.ClusterName,
			path.Join(path.Dir(assetsPath), "kubeconfig_"+asset.ClusterName))

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

func buildAsset(asset api.GKEAsset, builder *templates.Builder) (api.GKEAsset, error) {
	var err error
	var multiErr *multierror.Error

	asset.Credentials, err = builder.String(asset.Credentials)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build credentials"))
	asset.Project, err = builder.String(asset.Project)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build project"))

	asset.Region, err = builder.String(asset.Region)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build region"))

	asset.ClusterName, err = builder.String(asset.ClusterName)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build cluster_name"))

	asset.Zone, err = builder.String(asset.Zone)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build zone"))

	asset.InitialNodeCount, err = builder.String(asset.InitialNodeCount)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build initial_node_count"))

	asset.MachineType, err = builder.String(asset.MachineType)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build machine_type"))

	asset.AdditionalZones, err = builder.String(asset.AdditionalZones)
	multiErr = multierror.Append(multiErr, errors.Wrap(err, "build additional_zones"))

	// NOTE: items not configurable by the end user include MinMasterVersion
	return asset, multiErr.ErrorOrNil()
}

func renderTerraformContents(asset api.GKEAsset) (string, error) {
	var templateString string
	if shouldRenderProviderTempl(asset) {
		templateString += providerTempl
	}
	templateString += clusterTempl
	t, err := template.New("gkeTemplate").
		Funcs(sprig.TxtFuncMap()).
		Parse(templateString)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, asset)
}

func executeTemplate(t *template.Template, asset api.GKEAsset) (string, error) {
	var data = struct {
		api.GKEAsset
		KubeConfigTmpl string
	}{
		asset,
		kubeConfigTmpl,
	}
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

func shouldRenderProviderTempl(asset api.GKEAsset) bool {
	return asset.Credentials != "" || asset.Project != "" || asset.Region != ""
}
