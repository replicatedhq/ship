package specs

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/replicatedhq/ship/pkg/api"
	"gopkg.in/yaml.v2"
)

const baseSavePath = ".ship"

func getChartAndReadmeContents(ctx context.Context, chartPath string) error {
	splitPath := strings.Split(chartPath, "/")
	owner := splitPath[1]
	repo := splitPath[2]
	path := strings.Join(splitPath[3:], "/")

	client := github.NewClient(nil)
	_, dirContent, _, err := client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		return err
	}

	for _, gitContent := range dirContent {
		if gitContent.GetName() == "README.md" || gitContent.GetName() == "Chart.yaml" {
			downloadURL := gitContent.GetDownloadURL()
			savePath := filepath.Join(baseSavePath, gitContent.GetName())
			err := downloadFile(savePath, downloadURL)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadFile(path string, url string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (r *Resolver) resolveChartMetadata(ctx context.Context, path string) (api.HelmChartMetadata, error) {
	var md api.HelmChartMetadata

	err := getChartAndReadmeContents(ctx, path)
	if err != nil {
		return api.HelmChartMetadata{}, err
	}

	localChartPath := filepath.Join(baseSavePath, "Chart.yaml")
	chart, err := r.StateManager.FS.ReadFile(localChartPath)
	if err != nil {
		return api.HelmChartMetadata{}, err
	}

	if err := yaml.Unmarshal(chart, &md); err != nil {
		return api.HelmChartMetadata{}, err
	}

	return md, nil
}
