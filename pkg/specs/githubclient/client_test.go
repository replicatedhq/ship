package githubclient

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/google/go-github/v18/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/ship/pkg/constants"
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
	redirectArchive := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, serverURL+"/archive.tar.gz", http.StatusFound)
	}
	mux.HandleFunc("/repos/o/r/tarball/", redirectArchive)
	mux.HandleFunc("/repos/o/r/tarball", redirectArchive)

	mux.HandleFunc("/archive.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		archiveData := `H4sIAJKjXFsAA+3WXW6CQBQFYJbCBmrv/D831ce+uIOpDtGEKQaoibt3qERbEmiNI6TxfC8TIwkXTg65lfW73D3ZcrXZ7t1zcg9EZJRKv059OonL09lKmRDcMM6k0SkxSYolqbrLNB2fVW3LMIoPr2DounBZlg383z7H+fwnqp/5v25sWc8O1ucR7xHeh5ZyKH9xzl+TDPkroylJKeIMvR48//fw8PC4Ov1fLl7mb4uZX8e8xzX9V4Y1/RdMof9jyIpi6hFgQp3+1y78tLWrYm6CV+1/oum/JqGx/42hN/+12+XFwbuPsA7euA3++v1n/LL/sZA/JyM4vv9juMQ89SQwhd7+V67cb1fu5vInf9n/zLf+y6b/nDP0fwxtzFOPAQAAAAAAAAAAAACRHQEZehxJACgAAA==`
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(archiveData))
		w.Header().Set("Content-Type", "application/gzip")
		_, err := io.Copy(w, dec)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("GetFiles", func() {
		Context("With a url prefixed with http(s)", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithPrefix := "http://www.github.com/o/r/"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := &GithubClient{
					Client: client,
					Fs:     mockFs,
					Logger: log.NewNopLogger(),
				}

				dest, err := gitClient.GetFiles(context.Background(), validGitURLWithPrefix, constants.HelmChartPath)
				Expect(err).NotTo(HaveOccurred())

				readme, err := gitClient.Fs.ReadFile(path.Join(dest, "README.md"))
				Expect(err).NotTo(HaveOccurred())
				chart, err := gitClient.Fs.ReadFile(path.Join(dest, "Chart.yaml"))
				Expect(err).NotTo(HaveOccurred())
				deployment, err := gitClient.Fs.ReadFile(path.Join(dest, "templates", "deployment.yml"))
				Expect(err).NotTo(HaveOccurred())
				service, err := gitClient.Fs.ReadFile(path.Join(dest, "templates", "service.yml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
				Expect(string(deployment)).To(Equal("deployment"))
				Expect(string(service)).To(Equal("service"))
			})
		})

		Context("With a url not prefixed with http", func() {
			It("should fetch and persist README.md and Chart.yaml", func() {
				validGitURLWithoutPrefix := "github.com/o/r"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := &GithubClient{
					Client: client,
					Fs:     mockFs,
					Logger: log.NewNopLogger(),
				}

				dest, err := gitClient.GetFiles(context.Background(), validGitURLWithoutPrefix, constants.HelmChartPath)
				Expect(err).NotTo(HaveOccurred())

				readme, err := gitClient.Fs.ReadFile(path.Join(dest, "README.md"))
				Expect(err).NotTo(HaveOccurred())
				chart, err := gitClient.Fs.ReadFile(path.Join(dest, "Chart.yaml"))
				Expect(err).NotTo(HaveOccurred())
				deployment, err := gitClient.Fs.ReadFile(path.Join(dest, "templates", "deployment.yml"))
				Expect(err).NotTo(HaveOccurred())
				service, err := gitClient.Fs.ReadFile(path.Join(dest, "templates", "service.yml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(readme)).To(Equal("foo"))
				Expect(string(chart)).To(Equal("bar"))
				Expect(string(deployment)).To(Equal("deployment"))
				Expect(string(service)).To(Equal("service"))
			})
		})

		Context("With a non-github url", func() {
			It("should return an error", func() {
				nonGithubURL := "gitlab.com/o/r"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := &GithubClient{
					Client: client,
					Fs:     mockFs,
					Logger: log.NewNopLogger(),
				}

				_, err := gitClient.GetFiles(context.Background(), nonGithubURL, constants.HelmChartPath)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("http://gitlab.com/o/r is not a Github URL"))
			})
		})

		Context("With a url path to a single file at the base of the repo", func() {
			It("should fetch and persist the file", func() {
				validGithubURLSingle := "github.com/o/r/blob/master/Chart.yaml"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := &GithubClient{
					Client: client,
					Fs:     mockFs,
					Logger: log.NewNopLogger(),
				}

				dest, err := gitClient.GetFiles(context.Background(), validGithubURLSingle, constants.HelmChartPath)
				Expect(err).NotTo(HaveOccurred())

				chart, err := gitClient.Fs.ReadFile(filepath.Join(dest, "Chart.yaml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(chart)).To(Equal("bar"))
			})
		})

		Context("With a url path to a single nested file", func() {
			It("should fetch and persist the file", func() {
				validGithubURLSingle := "github.com/o/r/blob/master/templates/service.yml"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := &GithubClient{
					Client: client,
					Fs:     mockFs,
					Logger: log.NewNopLogger(),
				}

				dest, err := gitClient.GetFiles(context.Background(), validGithubURLSingle, constants.HelmChartPath)
				Expect(err).NotTo(HaveOccurred())
				chart, err := gitClient.Fs.ReadFile(filepath.Join(dest, "templates", "service.yml"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(chart)).To(Equal("service"))
			})
		})
	})

	Describe("decodeGitHubURL", func() {
		Context("With a valid github url", func() {
			It("should decode a valid url without a path", func() {
				chartPath := "github.com/o/r"
				o, r, b, p, err := decodeGitHubURL(chartPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(o).To(Equal("o"))
				Expect(r).To(Equal("r"))
				Expect(p).To(Equal(""))
				Expect(b).To(Equal(""))
			})

			It("should decode a valid url with a path", func() {
				chartPath := "github.com/o/r/stable/chart"
				o, r, b, p, err := decodeGitHubURL(chartPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(o).To(Equal("o"))
				Expect(r).To(Equal("r"))
				Expect(p).To(Equal("stable/chart"))
				Expect(b).To(Equal(""))
			})

			It("should decode a valid url with a /tree/<branch>/ path", func() {
				chartPath := "github.com/o/r/tree/master/stable/chart"
				o, r, b, p, err := decodeGitHubURL(chartPath)
				Expect(err).NotTo(HaveOccurred())

				Expect(o).To(Equal("o"))
				Expect(r).To(Equal("r"))
				Expect(p).To(Equal("stable/chart"))
				Expect(b).To(Equal("master"))
			})
		})

		Context("With an invalid github url", func() {
			It("should failed to decode a url without a path", func() {
				chartPath := "github.com"
				_, _, _, _, err := decodeGitHubURL(chartPath)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("github.com: unable to decode github url"))
			})

			It("should fail to decode a url with a path", func() {
				chartPath := "github.com/o"
				_, _, _, _, err := decodeGitHubURL(chartPath)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("github.com/o: unable to decode github url"))
			})
		})
	})

})

var _ = AfterSuite(func() {
	teardown()
})
