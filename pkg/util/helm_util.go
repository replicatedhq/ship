package util

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/helm"
	"github.com/spf13/afero"
)

// FetchUnpack fetches and unpacks the chart into a temp directory, then copies the contents of the chart folder to
// the destination dir.
// TODO figure out how to copy files from host into afero filesystem for testing, or how to force helm to fetch into afero
func FetchUnpack(chartRef, repoURL, version, dest, home string, filesystem afero.Afero) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to find the current working directory")
	}

	tmpDest, err := ioutil.TempDir(filepath.Join(wd, constants.ShipPath), "helm-fetch-unpack")
	if err != nil {
		return "", errors.Wrap(err, "unable to create temporary directory to unpack to")
	}
	defer os.RemoveAll(tmpDest)

	out, err := helm.Fetch(chartRef, repoURL, version, tmpDest, home)
	if err != nil {
		return out, err
	}

	subdir, err := FindOnlySubdir(tmpDest, filesystem)
	if err != nil {
		return "", errors.Wrap(err, "find chart subdir")
	}

	// rename that folder to move it to the destination directory
	err = os.Rename(subdir, dest)

	return "", err
}
