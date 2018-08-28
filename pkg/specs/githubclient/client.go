package githubclient

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
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type GithubClient struct {
	logger log.Logger
	client *github.Client
	fs     afero.Afero
}

func NewGithubClient(fs afero.Afero, logger log.Logger) *GithubClient {
	client := github.NewClient(nil)
	return &GithubClient{
		client: client,
		fs:     fs,
		logger: logger,
	}
}

func (g *GithubClient) GetFiles(
	ctx context.Context,
	upstream string,
	destinationPath string,
) error {
	debug := level.Debug(log.With(g.logger, "method", "getRepoContents"))

	if !strings.HasPrefix(upstream, "http") {

		upstream = fmt.Sprintf("http://%s", upstream)
	}

	debug.Log("event", "parseURL")
	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		return err
	}

	if !strings.Contains(upstreamURL.Host, "github.com") {
		return errors.New(fmt.Sprintf("%s is not a Github URL", upstream))
	}

	owner, repo, branch, repoPath, err := decodeGitHubURL(upstreamURL.Path)
	if err != nil {
		return err
	}

	debug.Log("event", "removeAll", "destinationPath", destinationPath)
	err = g.fs.RemoveAll(destinationPath)
	if err != nil {
		return errors.Wrap(err, "remove chart clone destination")
	}

	return g.downloadAndExtractFiles(ctx, owner, repo, branch, repoPath, destinationPath)
}

func (g *GithubClient) downloadAndExtractFiles(
	ctx context.Context,
	owner string,
	repo string,
	branch string,
	basePath string,
	filePath string,
) error {
	debug := level.Debug(log.With(g.logger, "method", "downloadAndExtractFiles"))

	debug.Log("event", "getContents", "path", basePath)

	archiveOpts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}
	archiveLink, _, err := g.client.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, archiveOpts)
	if err != nil {
		return errors.Wrapf(err, "get archive link for owner - %s repo - %s", owner, repo)
	}

	resp, err := http.Get(archiveLink.String())
	if err != nil {
		return errors.Wrapf(err, "downloading archive")
	}
	defer resp.Body.Close()

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "create uncompressed stream")
	}

	tarReader := tar.NewReader(uncompressedStream)

	basePathFound := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			if !basePathFound {
				branchString := branch
				if branchString == "" {
					branchString = "master"
				}
				return errors.Errorf("Path %s in %s/%s on branch %s not found", basePath, owner, repo, branchString)
			}
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
			basePathFound = true

			dirName = strings.TrimPrefix(dirName, basePath)
			if err := g.fs.MkdirAll(filepath.Join(filePath, dirName), 0755); err != nil {
				return errors.Wrapf(err, "extract tar gz, mkdir")
			}
		case tar.TypeReg:
			fileName := strings.Join(strings.Split(header.Name, "/")[1:], "/")
			if !strings.HasPrefix(fileName, basePath) {
				continue
			}
			basePathFound = true

			fileName = strings.TrimPrefix(fileName, basePath)
			outFile, err := g.fs.Create(filepath.Join(filePath, fileName))
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
