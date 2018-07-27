package specs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/go-kit/kit/log"

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
			}, {
			"type": "dir",
			"name": "templates",
			"download_url": "`+serverURL+`/fail"
			}]`)
	})
	mux.HandleFunc("/repos/o/r/contents/templates", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{
			"type": "file",
			"name": "deployment.yml",
			"download_url": "`+serverURL+`/download/deployment.yml"
			}, {
			"type": "file",
			"name": "service.yml",
			"download_url": "`+serverURL+`/download/service.yml"
			}]`)
	})
	mux.HandleFunc("/download/readme", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "foo")
	})
	mux.HandleFunc("/download/chart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "bar")
	})
	mux.HandleFunc("/download/deployment.yml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "deployment")
	})
	mux.HandleFunc("/download/service.yml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "service")
	})
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		Fail("should not try to download dirs")
	})

	Describe("GetChartAndReadmeContents", func() {
		Context("With a url prefixed with http(s)", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithPrefix := "http://www.github.com/o/r/"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
					logger: log.NewNopLogger(),
				}

				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithPrefix)

				readme, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "README.md"))
				Expect(err).NotTo(HaveOccurred())
				chart, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "Chart.yaml"))
				Expect(err).NotTo(HaveOccurred())
				deployment, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "templates", "deployment.yml"))
				Expect(err).NotTo(HaveOccurred())
				service, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "templates", "service.yml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
				Expect(string(deployment)).To(Equal("deployment"))
				Expect(string(service)).To(Equal("service"))
			})
		})

		Context("With a url not prefixed with http", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithoutPrefix := "github.com/o/r/"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
					logger: log.NewNopLogger(),
				}

				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithoutPrefix)
				readme, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "README.md"))
				Expect(err).NotTo(HaveOccurred())
				chart, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "Chart.yaml"))
				Expect(err).NotTo(HaveOccurred())
				deployment, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "templates", "deployment.yml"))
				Expect(err).NotTo(HaveOccurred())
				service, err := gitClient.fs.ReadFile(path.Join(constants.KustomizeHelmPath, "templates", "service.yml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
				Expect(string(deployment)).To(Equal("deployment"))
				Expect(string(service)).To(Equal("service"))
			})
		})
	})
})

var _ = AfterSuite(func() {
	teardown()
})
