package amazoneks

import (
	"bytes"
	"context"
	"html/template"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/inline"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Renderer is something that can render a terraform asset (that produces an EKS cluster) as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.EKSAsset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
		configGroups []libyaml.ConfigGroup,
	) func(ctx context.Context) error
}

// a LocalRenderer renders a terraform asset by writing generated terraform source code
type LocalRenderer struct {
	Logger log.Logger
	Inline inline.Renderer
	Fs     afero.Afero
}

var _ Renderer = &LocalRenderer{}

func NewRenderer(
	logger log.Logger,
	inline inline.Renderer,
	fs afero.Afero,
) Renderer {
	return &LocalRenderer{
		Logger: logger,
		Inline: inline,
		Fs:     fs,
	}
}

func (r *LocalRenderer) Execute(
	asset api.EKSAsset,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
	configGroups []libyaml.ConfigGroup,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		contents, err := renderTerraformContents(asset)
		if err != nil {
			return errors.Wrap(err, "render tf config")
		}

		assetsPath := "amazon_eks.tf"
		if asset.Dest != "" {
			assetsPath = asset.Dest
		}

		// save the path to the kubeconfig that running the generated terraform will produce
		templates.AddAmazonEKSPath(asset.ClusterName,
			path.Join(path.Dir(assetsPath), "kubeconfig_"+asset.ClusterName))

		assetsPath = path.Join(constants.InstallerPrefixPath, assetsPath)

		// write the inline spec
		err = r.Inline.Execute(
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

func renderTerraformContents(asset api.EKSAsset) (string, error) {
	templateString := ""
	if asset.CreatedVPC != nil {
		templateString = newVPCTempl
	} else if asset.ExistingVPC != nil {
		templateString = existingVPCTempl
	} else {
		return "", errors.New("a created or existing VPC must be provided")
	}

	templateString += workerTempl
	t, err := template.New("eksTemplate").Parse(templateString)
	if err != nil {
		return "", err
	}
	return executeTemplate(t, asset)
}

func executeTemplate(t *template.Template, asset api.EKSAsset) (string, error) {
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, asset); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
