package specs

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/go-kit/kit/log"

	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"

	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var client *github.Client
var mux *http.ServeMux
var serverURL string
var teardown func()

type ApplyUpstreamReleaseSpec struct {
	Name             string
	Description      string
	UpstreamShipYAML string
	ExpectedSpec     api.Spec
}

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
	mux.HandleFunc("/repos/o/r/tarball", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, serverURL+"/archive.tar.gz", http.StatusFound)
		return
	})
	mux.HandleFunc("/archive.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		archiveData := `H4sIAJKjXFsAA+3WXW6CQBQFYJbCBmrv/D831ce+uIOpDtGEKQaoibt3qERbEmiNI6TxfC8TIwkXTg65lfW73D3ZcrXZ7t1zcg9EZJRKv059OonL09lKmRDcMM6k0SkxSYolqbrLNB2fVW3LMIoPr2DounBZlg383z7H+fwnqp/5v25sWc8O1ucR7xHeh5ZyKH9xzl+TDPkroylJKeIMvR48//fw8PC4Ov1fLl7mb4uZX8e8xzX9V4Y1/RdMof9jyIpi6hFgQp3+1y78tLWrYm6CV+1/oum/JqGx/42hN/+12+XFwbuPsA7euA3++v1n/LL/sZA/JyM4vv9juMQ89SQwhd7+V67cb1fu5vInf9n/zLf+y6b/nDP0fwxtzFOPAQAAAAAAAAAAAACRHQEZehxJACgAAA==`
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(archiveData))
		w.Header().Set("Content-Type", "application/gzip")
		io.Copy(w, dec)
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

				err := gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithPrefix)
				Expect(err).NotTo(HaveOccurred())

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
				validGitURLWithoutPrefix := "github.com/o/r"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
					logger: log.NewNopLogger(),
				}

				err := gitClient.GetChartAndReadmeContents(context.Background(), validGitURLWithoutPrefix)
				Expect(err).NotTo(HaveOccurred())

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

		Context("With a non-github url", func() {
			It("should return an error", func() {
				nonGithubURL := "gitlab.com/o/r"
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				gitClient := GithubClient{
					client: client,
					fs:     mockFs,
					logger: log.NewNopLogger(),
				}

				err := gitClient.GetChartAndReadmeContents(context.Background(), nonGithubURL)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("http://gitlab.com/o/r is not a Github URL"))
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

	Describe("calculateContentSHA", func() {
		Context("With multiple files", func() {
			It("should calculate the same sha, multiple times", func() {
				mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
				mockFs.WriteFile("Chart.yaml", []byte("chart.yaml"), 0755)
				mockFs.WriteFile("templates/README.md", []byte("readme"), 0755)

				r := Resolver{
					FS: mockFs,
					StateManager: &state.MManager{
						Logger: log.NewNopLogger(),
						FS:     mockFs,
						V:      viper.New(),
					},
				}

				firstPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				secondPass, err := r.calculateContentSHA("")
				Expect(err).NotTo(HaveOccurred())

				Expect(firstPass).To(Equal(secondPass))
			})
		})
	})
})

var _ = AfterSuite(func() {
	teardown()
})

func TestResolveChartRelease(t *testing.T) {
	tests := []ApplyUpstreamReleaseSpec{
		{
			Name:         "no upstream",
			Description:  "no upstream, should use default release spec",
			ExpectedSpec: DefaultHelmRelease.Spec,
		},
		{
			Name:        "upstream exists",
			Description: "upstream exists, should use upstream release spec",
			UpstreamShipYAML: `
assets:
  v1: []
config:
  v1: []
lifecycle:
  v1:
   - helmIntro: {}
`,
			ExpectedSpec: api.Spec{
				Assets: api.Assets{
					V1: []api.Asset{},
				},
				Config: api.Config{
					V1: []libyaml.ConfigGroup{},
				},
				Lifecycle: api.Lifecycle{
					V1: []api.Step{
						{
							HelmIntro: &api.HelmIntro{},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			if test.UpstreamShipYAML != "" {
				mockFs.WriteFile(filepath.Join(constants.KustomizeHelmPath, "ship.yaml"), []byte(test.UpstreamShipYAML), 0755)
			}

			r := Resolver{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
				ui:     cli.NewMockUi(),
			}

			ctx := context.Background()
			spec, err := r.ResolveChartReleaseSpec(ctx)
			req.NoError(err)

			req.Equal(test.ExpectedSpec, spec)
		})
	}
}
