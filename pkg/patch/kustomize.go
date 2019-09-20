package patch

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/api/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/api/k8sdeps/validator"
	"sigs.k8s.io/kustomize/api/konfig"
	fLdr "sigs.k8s.io/kustomize/api/loader"
	"sigs.k8s.io/kustomize/api/plugins/builtins"
	pLdr "sigs.k8s.io/kustomize/api/plugins/loader"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/target"
)

func (p *ShipPatcher) RunKustomize(kustomizationPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	fsys := filesys.MakeFsOnDisk()

	if err := p.runKustomize(buf, fsys, kustomizationPath); err != nil {
		return nil, errors.Wrap(err, "failed to run kustomize build")
	}

	return buf.Bytes(), nil
}

func (p *ShipPatcher) runKustomize(out io.Writer, fSys filesys.FileSystem, kustomizationPath string) error {
	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return err
	}

	lrc := fLdr.RestrictionRootOnly
	ldr, err := fLdr.NewLoader(lrc, absPath, fSys)
	if err != nil {
		return errors.Wrap(err, "make loader")
	}
	// defer ldr.Cleanup()

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	pc, err := konfig.EnabledPluginConfig()
	if err != nil {
		return errors.Wrap(err, "make customized kustomize target")
	}
	kt, err := target.NewKustTarget(ldr, validator.NewKustValidator(), rf, transformer.NewFactoryImpl(), pLdr.NewLoader(pc, rf))
	if err != nil {
		return errors.Wrap(err, "make customized kustomize target")
	}

	allResources, err := kt.MakeCustomizedResMap()
	if err != nil {
		return errors.Wrap(err, "make customized res map")
	}

	err = builtins.NewLegacyOrderTransformerPlugin().Transform(allResources)
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
