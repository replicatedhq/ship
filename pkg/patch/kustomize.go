package patch

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/pkg/app"
	"sigs.k8s.io/kustomize/pkg/fs"
	"sigs.k8s.io/kustomize/pkg/loader"
)

func (p *ShipPatcher) RunKustomize(kustomizationPath string) ([]byte, error) {
	buf := new(bytes.Buffer)
	fsys := fs.MakeRealFS()

	if err := p.runKustomize(buf, fsys, kustomizationPath); err != nil {
		return nil, errors.Wrap(err, "failed to run kustomize build")
	}

	return buf.Bytes(), nil
}

// runKustomize is a repro of
// https://sigs.k8s.io/kustomize/blob/4569a09d54853003c5a474ab49a401a689bb58f6/pkg/commands/build.go#L72
func (p *ShipPatcher) runKustomize(out io.Writer, fSys fs.FileSystem, kustomizationPath string) error {
	l := loader.NewFileLoader(fSys)

	absPath, err := filepath.Abs(kustomizationPath)
	if err != nil {
		return err
	}

	rootLoader, err := l.New(absPath)
	if err != nil {
		return err
	}

	application, err := app.NewApplication(rootLoader, fSys)
	if err != nil {
		return err
	}

	allResources, err := application.MakeCustomizedResMap()

	if err != nil {
		return err
	}

	// Output the objects.
	res, err := allResources.EncodeAsYaml()
	if err != nil {
		return err
	}
	_, err = out.Write(res)
	return err
}
