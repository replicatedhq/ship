package patch

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/target"
	"sigs.k8s.io/kustomize/v3/pkg/validators"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
)

func (p *ShipPatcher) RunKustomize(kustomizationPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	fsys := fs.MakeRealFS()

	if err := p.runKustomize(buf, fsys, kustomizationPath); err != nil {
		return nil, errors.Wrap(err, "failed to run kustomize build")
	}

	return buf.Bytes(), nil
}

func (p *ShipPatcher) runKustomize(out io.Writer, fSys fs.FileSystem, kustomizationPath string) error {
	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return err
	}

	lrc := loader.RestrictionRootOnly
	ldr, err := loader.NewLoader(lrc, validators.MakeFakeValidator(), absPath, fSys)
	if err != nil {
		return errors.Wrap(err, "make loader")
	}
	// defer ldr.Cleanup()

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	pc := plugins.DefaultPluginConfig()
	kt, err := target.NewKustTarget(ldr, rf, transformer.NewFactoryImpl(), plugins.NewLoader(pc, rf))
	if err != nil {
		return errors.Wrap(err, "make customized kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return errors.Wrap(err, "make customized res map")
	}

	err = builtin.NewLegacyOrderTransformerPlugin().Transform(allResources)
	if err != nil {
		return errors.Wrap(err, "order res map")
	}

	// Output the objects.
	res, err := allResources.AsYaml()
	if err != nil {
		return errors.Wrap(err, "encode as yaml")
	}
	_, err = out.Write(res)
	return err
}
