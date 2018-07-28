package specs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	owner, repo, path, err := decodeGitHubUrl(chartURL.Path)
	if err != nil {
		return err
	}

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

	return g.downloadAndExtractFiles(ctx, owner, repo, path, "/")
}

func (g *GithubClient) downloadAndExtractFiles(ctx context.Context, owner string, repo string, basePath string, filePath string) error {
	debug := level.Debug(log.With(g.logger, "method", "downloadAndExtractFiles"))

	debug.Log("event", "getContents", "path", basePath)

	url, _, err := g.client.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, &github.RepositoryContentGetOptions{})
	if err != nil {
		return errors.Wrapf(err, "get archive link for owner - %s repo - %s", owner, repo)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return errors.Wrapf(err, "downloading archive")
	}
	defer resp.Body.Close()

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "create uncompressed stream")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrapf(err, "extract tar gz, next()")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirName := strings.Join(strings.Split(header.Name, "/")[1:], "/")
			if !strings.HasPrefix(dirName, basePath) {
				continue
			}

			dirName = strings.TrimPrefix(dirName, basePath)
			if err := g.fs.MkdirAll(filepath.Join(constants.KustomizeHelmPath, filePath, dirName), 0755); err != nil {
				return errors.Wrapf(err, "extract tar gz, mkdir")
			}
		case tar.TypeReg:
			fileName := strings.Join(strings.Split(header.Name, "/")[1:], "/")
			if !strings.HasPrefix(fileName, basePath) {
				continue
			}
			fileName = strings.TrimPrefix(fileName, basePath)
			outFile, err := g.fs.Create(filepath.Join(constants.KustomizeHelmPath, filePath, fileName))
			if err != nil {
				return errors.Wrapf(err, "extract tar gz, create")
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrapf(err, "extract tar gz, copy")
			}
		}
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

func decodeGitHubUrl(chartPath string) (string, string, string, error) {
	splitPath := strings.Split(chartPath, "/")

	if len(splitPath) < 3 {
		return "", "", "", errors.Wrapf(errors.New("unable to decode github url"), chartPath)
	}

	owner := splitPath[1]
	repo := splitPath[2]
	path := ""
	if len(splitPath) > 3 {
		path = strings.Join(splitPath[3:], "/")
	}

	return owner, repo, path, nil
}
