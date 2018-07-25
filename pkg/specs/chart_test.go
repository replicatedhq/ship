package specs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/replicatedhq/ship/pkg/constants"

	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var client *github.Client
var mux *http.ServeMux
var serverURL string
var teardown func()

func setupGitClient() (client *github.Client, mux *http.ServeMux, serveURL string, teardown func()) {
	mux = http.NewServeMux()
	server := httptest.NewServer(mux)
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url

	return client, mux, server.URL, server.Close
}

func TestGithubClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GithubClient")
}

var _ = Describe("GithubClient", func() {
	client, mux, serverURL, teardown = setupGitClient()
	mux.HandleFunc("/repos/o/r/contents/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
			"type": "file",
			"name": "README.md",
			"download_url": "`+serverURL+`/download/readme"
			}, {
			"type": "file",
			"name": "Chart.yaml",
			"download_url": "`+serverURL+`/download/chart"
			}]`)
	})
	mux.HandleFunc("/download/readme", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "foo")
	})
	mux.HandleFunc("/download/chart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "bar")
	})

	Describe("GetChartAndReadmeContents", func() {
		Context("With a url prefixed with http(s)", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithPrefix := "http://www.github.com/o/r/"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
				}

				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithPrefix)
				readme, err := gitClient.fs.ReadFile(path.Join(constants.BasePath, "README.md"))
				chart, err := gitClient.fs.ReadFile(path.Join(constants.BasePath, "Chart.yaml"))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
			})
		})

		Context("With a url not prefixed with http", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithoutPrefix := "github.com/o/r/"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
				}

				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithoutPrefix)
				readme, err := gitClient.fs.ReadFile(path.Join(constants.BasePath, "README.md"))
				chart, err := gitClient.fs.ReadFile(path.Join(constants.BasePath, "Chart.yaml"))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
			})
		})
	})
})

var _ = AfterSuite(func() {
	teardown()
})
