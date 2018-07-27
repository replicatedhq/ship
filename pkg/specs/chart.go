package specs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"

	"github.com/google/go-github/github"
	"github.com/replicatedhq/ship/pkg/api"
	"gopkg.in/yaml.v2"
)

type GithubClient struct {
	client *github.Client
	fs     afero.Afero
	logger log.Logger
}

func NewGithubClient(fs afero.Afero, logger log.Logger) *GithubClient {
	client := github.NewClient(nil)
	return &GithubClient{
		client: client,
		fs:     fs,
		logger: logger,
	}
}

func (g *GithubClient) GetChartAndReadmeContents(ctx context.Context, chartURLString string) error {
	debug := level.Debug(log.With(g.logger, "method", "getChartAndReadmeContents"))

	if !strings.HasPrefix(chartURLString, "http") {
		chartURLString = fmt.Sprintf("http://%s", chartURLString)
	}

	debug.Log("event", "parseURL")
	chartURL, err := url.Parse(chartURLString)
	if err != nil {
		return err
	}
	chartPath := chartURL.Path
	splitPath := strings.Split(chartPath, "/")
	owner := splitPath[1]
	repo := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	debug.Log("event", "checkExists", "path", constants.KustomizeHelmPath)
	saveDirExists, err := g.fs.Exists(constants.KustomizeHelmPath)
	if err != nil {
		return errors.Wrap(err, "check kustomizeHelmPath exists")
	}

	if saveDirExists {
		debug.Log("event", "removeAll", "path", constants.KustomizeHelmPath)
		err := g.fs.RemoveAll(constants.KustomizeHelmPath)
		if err != nil {
			return errors.Wrap(err, "remove kustomizeHelmPath")
		}
	}

	return g.getAllFiles(ctx, owner, repo, path, "/")
}

func (g *GithubClient) getAllFiles(ctx context.Context, owner string, repo string, basePath string, filePath string) error {
	debug := level.Debug(log.With(g.logger, "method", "getAllFiles"))

	debug.Log("event", "getContents", "path", basePath)
	_, dirContent, _, err := g.client.Repositories.GetContents(ctx, owner, repo, basePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		return errors.Wrapf(err, "initial get contents of owner - %s repo - %s", owner, repo)
	}

	for _, gitContent := range dirContent {
		if gitContent.GetType() == "file" {
			debug.Log("event", "git.download", "file", gitContent.GetName())
			savePath := filepath.Join(constants.KustomizeHelmPath, filePath)
			downloadURL := gitContent.GetDownloadURL()
			err := g.downloadFile(savePath, gitContent.GetName(), downloadURL)
			if err != nil {
				return errors.Wrapf(err, "download file %q", gitContent.GetName())
			}
		}
		if gitContent.GetType() == "dir" {
			debug.Log("event", "git.getAllFiles", "dir", gitContent.GetName())
			newBase := path.Join(basePath, gitContent.GetName())
			newFilePath := path.Join(filePath, gitContent.GetName())
			err := g.getAllFiles(ctx, owner, repo, newBase, newFilePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *GithubClient) downloadFile(path string, saveName string, url string) error {
	debug := level.Debug(log.With(g.logger, "method", "downloadFile"))

	debug.Log("event", "mkdir", "path", path)
	err := g.fs.MkdirAll(path, 0700)
	if err != nil {
		return err
	}

	debug.Log("event", "download", "url", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	debug.Log("event", "read.resp")
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	debug.Log("event", "write.file", "path", path)
	fullPath := filepath.Join(path, saveName)
	err = g.fs.WriteFile(fullPath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resolver) ResolveChartMetadata(ctx context.Context, path string) (api.HelmChartMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveChartMetadata"))

	debug.Log("phase", "fetch-readme", "for", path)
	var md api.HelmChartMetadata
	err := r.GithubClient.GetChartAndReadmeContents(ctx, path)
	if err != nil {
		return api.HelmChartMetadata{}, errors.Wrapf(err, "get chart and read me at %s", path)
	}

	localChartPath := filepath.Join(constants.KustomizeHelmPath, "Chart.yaml")
	debug.Log("phase", "read-chart", "from", localChartPath)
	chart, err := r.FS.ReadFile(localChartPath)
	if err != nil {
		return api.HelmChartMetadata{}, errors.Wrapf(err, "read file from %s", localChartPath)
	}

	localReadmePath := filepath.Join(constants.KustomizeHelmPath, "README.md")
	debug.Log("phase", "read-readme", "from", localReadmePath)
	readme, err := r.FS.ReadFile(localReadmePath)
	if err != nil {
		return api.HelmChartMetadata{}, errors.Wrapf(err, "read file from %s", localReadmePath)
	}

	debug.Log("phase", "unmarshal-chart.yaml")
	if err := yaml.Unmarshal(chart, &md); err != nil {
		return api.HelmChartMetadata{}, err
	}

	md.Readme = string(readme)
	return md, nil
}
