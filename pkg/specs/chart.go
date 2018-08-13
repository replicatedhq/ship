package specs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

	owner, repo, branch, path, err := decodeGitHubURL(chartURL.Path)
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

	return g.downloadAndExtractFiles(ctx, owner, repo, branch, path, "/")
}

func (g *GithubClient) downloadAndExtractFiles(ctx context.Context, owner string, repo string, branch string, basePath string, filePath string) error {
	debug := level.Debug(log.With(g.logger, "method", "downloadAndExtractFiles"))

	debug.Log("event", "getContents", "path", basePath)

	archiveOpts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}
	url, _, err := g.client.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, archiveOpts)
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

	debug.Log("phase", "save-chart-url", "url", path)
	md.URL = path

	debug.Log("phase", "calculate-sha", "for", constants.KustomizeHelmPath)
	contentSHA, err := r.calculateContentSHA(constants.KustomizeHelmPath)
	if err != nil {
		return api.HelmChartMetadata{}, errors.Wrapf(err, "calculate chart sha")
	}
	md.ContentSHA = contentSHA

	localChartPath := filepath.Join(constants.KustomizeHelmPath, "Chart.yaml")
	debug.Log("phase", "read-chart", "from", localChartPath)
	chart, err := r.FS.ReadFile(localChartPath)
	if err != nil {
		r.ui.Error(
			"The input was not recognized as a supported asset type. Ship currently supports Helm, Knative, and Kubernetes applications. Check the URL and try again.\n",
		)
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

func (r *Resolver) ResolveChartRelease(ctx context.Context) (api.Release, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveChartRelease"))

	localReleasePath := filepath.Join(constants.KustomizeHelmPath, "ship.yaml")

	debug.Log("phase", "read-release", "from", localReleasePath)
	var upstreamRelease api.Release
	release, err := r.FS.ReadFile(localReleasePath)
	if err != nil {
		level.Debug(log.With(r.Logger, "event", "read file from %s", localReleasePath))
		return api.Release{}, nil
	}

	debug.Log("phase", "unmarshal ship.yaml", "from", localReleasePath)
	if err := json.Unmarshal(release, &upstreamRelease); err == nil {
		level.Debug(log.With(r.Logger, "event", "unmarshal release from %s", localReleasePath))
		return api.Release{}, errors.Wrapf(err, "unmarshal ship.yaml")
	}

	return upstreamRelease, nil
}

func (r *Resolver) calculateContentSHA(root string) (string, error) {
	contents := []byte{}
	err := r.FS.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "fs walk")
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fileContents, err := r.FS.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read file")
		}

		contents = append(contents, fileContents...)
		return nil
	})

	if err != nil {
		return "", errors.Wrapf(err, "calculate content sha")
	}

	return fmt.Sprintf("%x", sha256.Sum256(contents)), nil
}

func decodeGitHubURL(chartPath string) (owner string, repo string, branch string, path string, err error) {
	splitPath := strings.Split(chartPath, "/")

	if len(splitPath) < 3 {
		return owner, repo, path, branch, errors.Wrapf(errors.New("unable to decode github url"), chartPath)
	}

	owner = splitPath[1]
	repo = splitPath[2]
	branch = ""
	path = ""
	if len(splitPath) > 3 {
		if splitPath[3] == "tree" {
			branch = splitPath[4]
			path = strings.Join(splitPath[5:], "/")
		} else {
			path = strings.Join(splitPath[3:], "/")
		}
	}

	return owner, repo, branch, path, nil
}
