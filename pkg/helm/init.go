/*
Copyright The Helm Authors.

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

/*
This file was edited by Replicated in 2018 to remove some functionality and to expose `helm init` as a function.
Among other things, clientOnly has been set as the default (and only) option, and the cli interface code has been removed.
*/

package helm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/cmd/helm/installer"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

// const initDesc = `
// This command installs Tiller (the Helm server-side component) onto your
// Kubernetes Cluster and sets up local configuration in $HELM_HOME (default ~/.helm/).
//
// As with the rest of the Helm commands, 'helm init' discovers Kubernetes clusters
// by reading $KUBECONFIG (default '~/.kube/config') and using the default context.
//
// To set up just a local environment, use '--client-only'. That will configure
// $HELM_HOME, but not attempt to connect to a Kubernetes cluster and install the Tiller
// deployment.
//
// When installing Tiller, 'helm init' will attempt to install the latest released
// version. You can specify an alternative image with '--tiller-image'. For those
// frequently working on the latest code, the flag '--canary-image' will install
// the latest pre-release version of Tiller (e.g. the HEAD commit in the GitHub
// repository on the master branch).
//
// To dump a manifest containing the Tiller deployment YAML, combine the
// '--dry-run' and '--debug' flags.
// `

const (
	stableRepository         = "stable"
	localRepository          = "local"
	localRepositoryIndexFile = "index.yaml"
)

var (
	stableRepositoryURL = "https://kubernetes-charts.storage.googleapis.com"
	// This is the IPv4 loopback, not localhost, because we have to force IPv4
	// for Dockerized Helm: https://github.com/kubernetes/helm/issues/1410
	localRepositoryURL = "http://127.0.0.1:8879/charts"
)

type initCmd struct {
	skipRefresh bool // do not refresh (download) the local repository cache
	out         io.Writer
	home        helmpath.Home
	opts        installer.Options
}

func Init(home string) (string, error) {
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	toInit := initCmd{
		out: bufWriter,
	}

	var path string
	var err error
	if home != "" {
		path, err = filepath.Abs(home)
	} else {
		path, err = helmHome()
	}
	if err != nil {
		return "", errors.Wrap(err, "unable to find home directory")
	}
	toInit.home = helmpath.Home(path)

	err = toInit.run()
	return buf.String(), err
}

// run initializes local config and installs Tiller to Kubernetes cluster.
func (i *initCmd) run() error {

	writeYAMLManifests := func(manifests []string) error {
		w := i.out
		for _, manifest := range manifests {
			if _, err := fmt.Fprintln(w, "---"); err != nil {
				return err
			}

			if _, err := fmt.Fprintln(w, manifest); err != nil {
				return err
			}
		}

		// YAML ending document boundary marker
		_, err := fmt.Fprintln(w, "...")
		return err
	}
	if len(i.opts.Output) > 0 {
		var manifests []string
		var err error
		if manifests, err = installer.TillerManifests(&i.opts); err != nil {
			return err
		}
		switch i.opts.Output.String() {
		case "json":
			for _, manifest := range manifests {
				var out bytes.Buffer
				jsonb, err := yaml.ToJSON([]byte(manifest))
				if err != nil {
					return err
				}
				buf := bytes.NewBuffer(jsonb)
				if err := json.Indent(&out, buf.Bytes(), "", "    "); err != nil {
					return err
				}
				if _, err = i.out.Write(out.Bytes()); err != nil {
					return err
				}
				fmt.Fprint(i.out, "\n")
			}
			return nil
		case "yaml":
			return writeYAMLManifests(manifests)
		default:
			return fmt.Errorf("unknown output format: %q", i.opts.Output)
		}
	}
	if settings.Debug {
		var manifests []string
		var err error

		// write Tiller manifests
		if manifests, err = installer.TillerManifests(&i.opts); err != nil {
			return err
		}

		if err = writeYAMLManifests(manifests); err != nil {
			return err
		}
	}

	if err := ensureDirectories(i.home, i.out); err != nil {
		return err
	}
	if err := ensureDefaultRepos(i.home, i.out, i.skipRefresh); err != nil {
		return err
	}
	if err := ensureRepoFileFormat(i.home.RepositoryFile(), i.out); err != nil {
		return err
	}
	fmt.Fprintf(i.out, "$HELM_HOME has been configured at %s.\n", i.home)

	fmt.Fprintln(i.out, "Happy Helming!")
	return nil
}

// ensureDirectories checks to see if $HELM_HOME exists.
//
// If $HELM_HOME does not exist, this function will create it.
func ensureDirectories(home helmpath.Home, out io.Writer) error {
	configDirectories := []string{
		home.String(),
		home.Repository(),
		home.Cache(),
		home.LocalRepository(),
		home.Plugins(),
		home.Starters(),
		home.Archive(),
	}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			fmt.Fprintf(out, "Creating %s \n", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("Could not create %s: %s", p, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}

	return nil
}

func ensureDefaultRepos(home helmpath.Home, out io.Writer, skipRefresh bool) error {
	repoFile := home.RepositoryFile()
	if fi, err := os.Stat(repoFile); err != nil {
		fmt.Fprintf(out, "Creating %s \n", repoFile)
		f := repo.NewRepoFile()
		sr, err := initStableRepo(home.CacheIndex(stableRepository), out, skipRefresh, home)
		if err != nil {
			return err
		}
		lr, err := initLocalRepo(home.LocalRepository(localRepositoryIndexFile), home.CacheIndex("local"), out, home)
		if err != nil {
			return err
		}
		f.Add(sr)
		f.Add(lr)
		if err := f.WriteFile(repoFile, 0644); err != nil {
			return err
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", repoFile)
	}
	return nil
}

func initStableRepo(cacheFile string, out io.Writer, skipRefresh bool, home helmpath.Home) (*repo.Entry, error) {
	fmt.Fprintf(out, "Adding %s repo with URL: %s \n", stableRepository, stableRepositoryURL)
	c := repo.Entry{
		Name:  stableRepository,
		URL:   stableRepositoryURL,
		Cache: cacheFile,
	}
	r, err := repo.NewChartRepository(&c, getter.All(environment.EnvSettings{Home: home}))
	if err != nil {
		return nil, err
	}

	if skipRefresh {
		return &c, nil
	}

	// In this case, the cacheFile is always absolute. So passing empty string
	// is safe.
	if err := r.DownloadIndexFile(""); err != nil {
		return nil, fmt.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", stableRepositoryURL, err.Error())
	}

	return &c, nil
}

func initLocalRepo(indexFile, cacheFile string, out io.Writer, home helmpath.Home) (*repo.Entry, error) {
	if fi, err := os.Stat(indexFile); err != nil {
		fmt.Fprintf(out, "Adding %s repo with URL: %s \n", localRepository, localRepositoryURL)
		i := repo.NewIndexFile()
		if err := i.WriteFile(indexFile, 0644); err != nil {
			return nil, err
		}

		//TODO: take this out and replace with helm update functionality
		if err := createLink(indexFile, cacheFile, home); err != nil {
			return nil, err
		}
	} else if fi.IsDir() {
		return nil, fmt.Errorf("%s must be a file, not a directory", indexFile)
	}

	return &repo.Entry{
		Name:  localRepository,
		URL:   localRepositoryURL,
		Cache: cacheFile,
	}, nil
}

func ensureRepoFileFormat(file string, out io.Writer) error {
	r, err := repo.LoadRepositoriesFile(file)
	if err == repo.ErrRepoOutOfDate {
		fmt.Fprintln(out, "Updating repository file format...")
		if err := r.WriteFile(file, 0644); err != nil {
			return err
		}
	}

	return nil
}
