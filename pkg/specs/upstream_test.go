package specs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v18/github"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
)

func setupGitClient() (client *github.Client, mux *http.ServeMux, serveURL string, teardown func()) {
	mux = http.NewServeMux()
	server := httptest.NewServer(mux)
	client = github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url
	client.UploadURL = url

	return client, mux, server.URL, server.Close
}

func TestResolver_MaybeResolveVersionedUpstream(t *testing.T) {
	tests := []struct {
		name         string
		upstream     string
		currentState state.State
		expected     string
		expectErr    bool
	}{
		{
			name:     "versioned upstream has an update available",
			upstream: "github.com/o/r/tree/_latest_",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "1.1.0",
					},
				},
			},
			expected: "github.com/o/r/tree/1.2.0",
		},
		{
			name:     "version upstream is above latest",
			upstream: "github.com/o/r/tree/_latest_",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "1.2.1",
					},
				},
			},
			expected:  "",
			expectErr: true,
		},
		{
			name:     "commit sha upstream",
			upstream: "github.com/o/r/tree/d3eed9a347ad02f0b79e3f92330878f88953cf64/path",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "1.2.0",
					},
				},
			},
			expected: "github.com/o/r/tree/d3eed9a347ad02f0b79e3f92330878f88953cf64/path",
		},
		{
			name:     "ref upstream",
			upstream: "github.com/o/r/tree/abcedfg/path",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "1.2.0",
					},
				},
			},
			expected: "github.com/o/r/tree/abcedfg/path",
		},
		{
			name:     "versioned upstream with no latest release",
			upstream: "github.com/a/b/tree/_latest_",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "1.2.0",
					},
				},
			},
			expected:  "",
			expectErr: true,
		},
		{
			name:     "versioned upstream with no version in state",
			upstream: "github.com/o/r/tree/_latest_",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{},
				},
			},
			expected: "github.com/o/r/tree/1.2.0",
		},
		{
			name:     "ref upstream with no version in state",
			upstream: "github.com/o/r/tree/abranch",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{},
				},
			},
			expected: "github.com/o/r/tree/abranch",
		},
		{
			name:     "not a github url",
			upstream: "notgithub.com/o/r/tree/_latest_",
			currentState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{},
				},
			},
			expected: "notgithub.com/o/r/tree/_latest_",
		},
	}

	client, mux, _, teardown := setupGitClient()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		latestTag := "1.2.0"
		latest := struct {
			TagName *string `json:"tag_name"`
		}{
			TagName: &latestTag,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(latest)
	})
	defer teardown()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testLogger := &logger.TestLogger{T: t}
			req := require.New(t)
			r := &Resolver{
				GitHubFetcher: &githubclient.GithubClient{
					Client: client,
					Logger: testLogger,
				},
				Logger: testLogger,
			}
			p := r.NewContentProcessor()
			actual, err := p.MaybeResolveVersionedUpstream(context.Background(), tt.upstream, tt.currentState)
			if tt.expectErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(tt.expected, actual)
		})
	}
}

func TestResolver_maybeCreateVersionedUpstream(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
		expected string
	}{
		{
			name:     "no version - no change",
			upstream: "github.com/helm/charts/stable/grafana",
			expected: "github.com/helm/charts/stable/grafana",
		},
		{
			name:     "has a version - changed",
			upstream: "https://github.com/istio/istio/tree/1.0.2/install/kubernetes/helm/istio",
			expected: "https://github.com/istio/istio/tree/_latest_/install/kubernetes/helm/istio",
		},
		{
			name:     "has a ref - no changed",
			upstream: "https://github.com/istio/istio/tree/abranch/install/kubernetes/helm/istio",
			expected: "https://github.com/istio/istio/tree/abranch/install/kubernetes/helm/istio",
		},
		{
			name:     "not a github url - no change",
			upstream: "some-website.com/chart/chart/tree/1.0.2",
			expected: "some-website.com/chart/chart/tree/1.0.2",
		},
		{
			name:     "github url with correct format but malformed version - no change",
			upstream: "github.com/owner/repo/tree/b0asdf",
			expected: "github.com/owner/repo/tree/b0asdf",
		},
		{
			name:     "github url with branch ref - no change",
			upstream: "github.com/owner/repo/tree/branch",
			expected: "github.com/owner/repo/tree/branch",
		},
		{
			name:     "github url with latest token - no change",
			upstream: "https://github.com/istio/istio/tree/_latest_/install/kubernetes/helm/istio",
			expected: "https://github.com/istio/istio/tree/_latest_/install/kubernetes/helm/istio",
		},
		{
			name:     "github url with commit sha - no change",
			upstream: "https://github.com/o/r/tree/507feecae588c958ebe82bcf701b8be63f34ac9b",
			expected: "https://github.com/o/r/tree/507feecae588c958ebe82bcf701b8be63f34ac9b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			r := &Resolver{
				Logger: &logger.TestLogger{T: t},
			}
			actual, err := r.maybeCreateVersionedUpstream(tt.upstream)
			req.NoError(err)

			req.Equal(tt.expected, actual)
		})
	}
}
