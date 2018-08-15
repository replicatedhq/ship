package helm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
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

	devel bool // use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored.

	out io.Writer
}

func Fetch(chartRef, repoURL, version, dest string) (string, error) {

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

	err := toFetch.run()
	return buf.String(), err
}

func (f *fetchCmd) run() error {
	c := downloader.ChartDownloader{
		Out:      f.out,
		Keyring:  f.keyring,
		Verify:   downloader.VerifyNever,
		Getters:  getter.All(environment.EnvSettings{}),
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
		chartURL, err := repo.FindChartInAuthRepoURL(f.repoURL, f.username, f.password, f.chartRef, f.version, f.certFile, f.keyFile, f.caFile, getter.All(environment.EnvSettings{}))
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
