package specs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"

	"github.com/google/go-github/github"
	"github.com/replicatedhq/ship/pkg/api"
	"gopkg.in/yaml.v2"
)

type GithubClient struct {
	client *github.Client
	fs     afero.Afero
}

func NewGithubClient(fs afero.Afero) *GithubClient {
	client := github.NewClient(nil)
	return &GithubClient{
		client: client,
		fs:     fs,
	}
}

func (g GithubClient) GetChartAndReadmeContents(ctx context.Context, chartURLString string) error {
	if !strings.HasPrefix(chartURLString, "http") {
		chartURLString = fmt.Sprintf("http://%s", chartURLString)
	}

	chartURL, err := url.Parse(chartURLString)
	chartPath := chartURL.Path
	splitPath := strings.Split(chartPath, "/")
	owner := splitPath[1]
	repo := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	_, dirContent, _, err := g.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		return err
	}

	for _, gitContent := range dirContent {
		if gitContent.GetName() == "README.md" || gitContent.GetName() == "Chart.yaml" {
			downloadURL := gitContent.GetDownloadURL()
			savePath := filepath.Join(constants.BasePath, gitContent.GetName())

			err := g.downloadFile(savePath, downloadURL)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (g GithubClient) downloadFile(path string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = g.fs.WriteFile(path, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resolver) resolveChartMetadata(ctx context.Context, path string) (api.HelmChartMetadata, error) {
	var md api.HelmChartMetadata
	err := r.GithubClient.GetChartAndReadmeContents(ctx, path)
	if err != nil {
		return api.HelmChartMetadata{}, err
	}

	localChartPath := filepath.Join(constants.BasePath, "Chart.yaml")
	chart, err := r.StateManager.FS.ReadFile(localChartPath)
	if err != nil {
		return api.HelmChartMetadata{}, err
	}

	if err := yaml.Unmarshal(chart, &md); err != nil {
		return api.HelmChartMetadata{}, err
	}

	return md, nil
}
