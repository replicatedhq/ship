package helm

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/replicatedhq/ship/pkg/constants"

	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/gogetter"
	"github.com/replicatedhq/ship/pkg/util"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"k8s.io/helm/pkg/chartutil"
)

func (f *LocalTemplater) addDependencies(
	dependencies []*chartutil.Dependency,
	helmHome string,
	chartRoot string,
	asset api.HelmAsset,
) error {
	for _, dependency := range dependencies {
		repoURL, err := url.Parse(dependency.Repository)
		if err != nil {
			return errors.Wrapf(err, "parse dependency repo %s", dependency.Repository)
		}
		if repoURL.Scheme == "file" {
			if err := f.getLocalDependency(dependency.Repository, chartRoot, asset, helmHome); err != nil {
				return errors.Wrapf(err, "get local dep %s", dependency.Repository)
			}
		} else {
			repoName := strings.Split(repoURL.Hostname(), ".")[0]
			if err := f.Commands.RepoAdd(repoName, dependency.Repository, helmHome); err != nil {
				return errors.Wrapf(err, "add helm repo %s", dependency.Repository)
			}
		}
	}

	return nil
}

func (f *LocalTemplater) getLocalDependency(repo string, chartRoot string, originalAsset api.HelmAsset, helmHome string) error {
	var depPath string
	var err error
	p := strings.TrimPrefix(repo, "file://")

	// root path is absolute
	if strings.HasPrefix(p, "/") {
		if depPath, err = filepath.Abs(p); err != nil {
			return err
		}
	} else {
		depPath = filepath.Join(chartRoot, p)
	}

	depPathExists, err := f.FS.DirExists(depPath)
	if err != nil || !depPathExists {
		depUpstream, err := f.createDependencyUpstreamFromAsset(originalAsset, p)
		if err != nil {
			return errors.Wrap(err, "create dependency upstream")
		}
		savedPath, err := f.fetchLocalHelmDependency(depUpstream, constants.HelmLocalDependencyPath)
		if err != nil {
			return errors.Wrap(err, "fetch local helm dependency")
		}
		if err := f.FS.MkdirAll(filepath.Dir(depPath), 0755); err != nil {
			return errors.Wrap(err, "mkdirall dep path")
		}
		if err := f.FS.Rename(savedPath, depPath); err != nil {
			return errors.Wrap(err, "rename to dep path")
		}
		if err := f.FS.RemoveAll(constants.HelmLocalDependencyPath); err != nil {
			return errors.Wrap(err, "remove tmp local helm dependency")
		}
	}

	return nil
}

// Local path creation logic taken from
// https://github.com/helm/helm/blob/c82c0b6046b852f449dcaae768ba57331116dc87/pkg/resolver/resolver.go#L132
func (f *LocalTemplater) getLocalDependencyPath(repo string, chartRoot string) (string, error) {
	var depPath string
	var err error
	p := strings.TrimPrefix(repo, "file://")

	// root path is absolute
	if strings.HasPrefix(p, "/") {
		if depPath, err = filepath.Abs(p); err != nil {
			return "", err
		}
	} else {
		depPath = filepath.Join(chartRoot, p)
	}

	return depPath, nil
}

func (f *LocalTemplater) createDependencyUpstreamFromAsset(originalAsset api.HelmAsset, path string) (string, error) {
	upstream := originalAsset.Upstream
	if util.IsGithubURL(upstream) {
		githubURL, err := util.ParseGithubURL(upstream, "master")
		if err != nil {
			return "", errors.Wrap(err, "parse github url")
		}

		depPath := filepath.Join(githubURL.Subdir, path)
		githubURL.Subdir = depPath
		return githubURL.URL(), nil
	}

	return "", nil
}

func (f *LocalTemplater) fetchLocalHelmDependency(upstream string, fetchPath string) (string, error) {
	var fetcher apptype.FileFetcher
	fetcher = githubclient.NewGithubClient(f.FS, f.Logger)
	if f.Viper.GetBool("prefer-git") {
		var isSingleFile bool
		var subdir string
		upstream, subdir, isSingleFile = gogetter.UntreeGithub(upstream)
		fetcher = &gogetter.GoGetter{Logger: f.Logger, FS: f.FS, Subdir: subdir, IsSingleFile: isSingleFile}
	}

	savedPath, err := fetcher.GetFiles(context.Background(), upstream, fetchPath)
	if err != nil {
		return "", errors.Wrap(err, "get files")
	}

	return savedPath, nil
}

func (f *LocalTemplater) getDependencyPath(chartRoot, path string) string {
	return filepath.Join(chartRoot, path)
}
