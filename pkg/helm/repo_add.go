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

/*This file was edited by Replicated in 2018 to
  - expose `helm repo add` as a function
  - silence the error output from the cobra command.
*/

package helm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"syscall"

	"golang.org/x/crypto/ssh/terminal"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

type repoAddCmd struct {
	name     string
	url      string
	username string
	password string
	home     helmpath.Home
	noupdate bool

	certFile string
	keyFile  string
	caFile   string

	out io.Writer
}

func RepoAdd(name, url, home string) (string, error) {
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	toRepoAdd := repoAddCmd{
		name: name,
		url:  url,

		out: bufWriter,
	}

	if home != "" {
		toRepoAdd.home = helmpath.Home(home)
	} else {
		path, err := helmHome()
		if err != nil {
			return "", errors.Wrap(err, "unable to find home directory")
		}
		toRepoAdd.home = helmpath.Home(path)
	}

	err := toRepoAdd.run()
	return buf.String(), err
}

func (a *repoAddCmd) run() error {
	if a.username != "" && a.password == "" {
		fmt.Fprint(a.out, "Password:")
		password, err := readPassword()
		fmt.Fprintln(a.out)
		if err != nil {
			return err
		}
		a.password = password
	}

	if err := addRepository(a.name, a.url, a.username, a.password, a.home, a.certFile, a.keyFile, a.caFile, a.noupdate); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "%q has been added to your repositories\n", a.name)
	return nil
}

func readPassword() (string, error) {
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func addRepository(name, url, username, password string, home helmpath.Home, certFile, keyFile, caFile string, noUpdate bool) error {
	f, err := repo.LoadRepositoriesFile(home.RepositoryFile())
	if err != nil {
		return err
	}

	if noUpdate && f.Has(name) {
		return fmt.Errorf("repository name (%s) already exists, please specify a different name", name)
	}

	cif := home.CacheIndex(name)
	c := repo.Entry{
		Name:     name,
		Cache:    cif,
		URL:      url,
		Username: username,
		Password: password,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return err
	}

	if err := r.DownloadIndexFile(home.Cache()); err != nil {
		return fmt.Errorf("Looks like %q is not a valid chart repository or cannot be reached: %s", url, err.Error())
	}

	f.Update(&c)

	return f.WriteFile(home.RepositoryFile(), 0644)
}
