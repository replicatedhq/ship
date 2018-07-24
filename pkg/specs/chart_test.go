package specs

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testSavePath = ".test"

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

var _ = BeforeSuite(func() {
	os.Mkdir(testSavePath, 0700)
})

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
				gitClient := GithubClient{
					client:   client,
					savePath: testSavePath,
				}
				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithPrefix)
				readme, err := ioutil.ReadFile(path.Join(testSavePath, "README.md"))
				chart, err := ioutil.ReadFile(path.Join(testSavePath, "Chart.yaml"))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
			})
		})

		Context("With a url not prefixed with http", func() {
			It("should be a short story", func() {
				Expect(true).To(Equal(true))
				validGitURLWithoutPrefix := "github.com/o/r/"
				gitClient := GithubClient{
					client:   client,
					savePath: testSavePath,
				}
				gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithoutPrefix)
				readme, err := ioutil.ReadFile(path.Join(testSavePath, "README.md"))
				chart, err := ioutil.ReadFile(path.Join(testSavePath, "Chart.yaml"))

				Expect(err).NotTo(HaveOccurred())
				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
			})
		})
	})
})

var _ = AfterSuite(func() {
	teardown()
	os.RemoveAll(testSavePath)
})
