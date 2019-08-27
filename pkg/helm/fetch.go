/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*This file was edited by Replicated in 2018 to
  - expose `helm fetch` as a function
  - silence the error output from the cobra command.
  - remove some functionality (todo document this)
*/

package helm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

type fetchCmd struct {
	chartRef string // chart URL | repo/chartname
	destdir  string // location to write the chart. If this and tardir are specified, tardir is appended to this
	version  string // specific version of a chart. Without this, the latest version is fetched
	repoURL  string // chart repository url where to locate the requested chart
	username string // chart repository username
	password string // chart repository password

	verify      bool   // verify the package against its signature
	verifyLater bool   // fetch the provenance file, but don't perform verification
	keyring     string // keyring containing public keys

	certFile string // identify HTTPS client using this SSL certificate file
	keyFile  string // identify HTTPS client using this SSL key file
	caFile   string // verify certificates of HTTPS-enabled servers using this CA bundle

	// devel bool // use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored.

	out io.Writer

	home helmpath.Home // helm home directory
}

func Fetch(chartRef, repoURL, version, dest, home string) (string, error) {

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	toFetch := fetchCmd{
		chartRef: chartRef,
		repoURL:  repoURL,
		version:  version,
		destdir:  dest,

		keyring: defaultKeyring(),
		out:     bufWriter,
	}

	if home != "" {
		toFetch.home = helmpath.Home(home)
	} else {
		path, err := helmHome()
		if err != nil {
			return "", errors.Wrap(err, "unable to find home directory")
		}
		toFetch.home = helmpath.Home(path)
	}

	err := toFetch.run()
	return buf.String(), err
}

func (f *fetchCmd) run() error {
	c := downloader.ChartDownloader{
		HelmHome: f.home,
		Out:      f.out,
		Keyring:  f.keyring,
		Verify:   downloader.VerifyNever,
		Getters:  getter.All(environment.EnvSettings{Home: f.home}),
		Username: f.username,
		Password: f.password,
	}

	if f.verify {
		c.Verify = downloader.VerifyAlways
	} else if f.verifyLater {
		c.Verify = downloader.VerifyLater
	}

	// we fetch to a tempdir, then untar and copy
	dest, err := ioutil.TempDir("", "helm-")
	if err != nil {
		return fmt.Errorf("Failed to untar: %s", err)
	}
	defer os.RemoveAll(dest)

	if f.repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(f.repoURL, f.username, f.password, f.chartRef, f.version, f.certFile, f.keyFile, f.caFile, getter.All(environment.EnvSettings{Home: f.home}))
		if err != nil {
			return err
		}
		f.chartRef = chartURL
	}

	saved, v, err := c.DownloadTo(f.chartRef, f.version, dest)
	if err != nil {
		return err
	}

	if f.verify {
		fmt.Fprintf(f.out, "Verification: %v\n", v)
	}

	// untar the chart into the requested directory.
	if fi, err := os.Stat(f.destdir); err != nil {
		if err := os.MkdirAll(f.destdir, 0755); err != nil {
			return fmt.Errorf("Failed to untar (mkdir): %s", err)
		}

	} else if !fi.IsDir() {
		return fmt.Errorf("Failed to untar: %s is not a directory", f.destdir)
	}

	return chartutil.ExpandFile(f.destdir, saved)
}

// defaultKeyring returns the expanded path to the default keyring.
func defaultKeyring() string {
	return os.ExpandEnv("$HOME/.gnupg/pubring.gpg")
}
