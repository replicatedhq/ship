package specs

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	replicatedapp2 "github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/test-mocks/apptype"
	"github.com/replicatedhq/ship/pkg/test-mocks/githubclient"
	"github.com/replicatedhq/ship/pkg/test-mocks/replicatedapp"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/ui"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestResolver_ResolveRelease(t *testing.T) {
	viperResolver := Resolver{Viper: viper.New()}
	ctx := context.Background()
	tests := []struct {
		name      string
		upstream  string
		shaSummer shaSummer
		expect    func(
			t *testing.T,
			mockUi *ui.MockUi,
			appType *apptype.MockInspector,
			mockState *state.MockManager,
			mockFs afero.Afero,
			mockAppResolver *replicatedapp.MockResolver,
			mockReleaseNotesFetcher *githubclient.MockGitHubReleaseNotesFetcher,
		)
		expectRelease *api.Release
	}{
		{
			name:     "helm chart in github",
			upstream: "github.com/helm/charts/stable/x5",
			shaSummer: func(resolver *Resolver, s string) (string, error) {
				return "abcdef1234567890", nil
			},
			expect: func(
				t *testing.T,
				mockUi *ui.MockUi,
				appType *apptype.MockInspector,
				mockState *state.MockManager,
				mockFs afero.Afero,
				mockAppResolver *replicatedapp.MockResolver,
				mockReleaseNotesFetcher *githubclient.MockGitHubReleaseNotesFetcher,
			) {
				req := require.New(t)
				inOrder := mockUi.EXPECT().Info("Reading github.com/helm/charts/stable/x5 ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "github.com/helm/charts/stable/x5").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						err := mockFs.MkdirAll("fake-tmp", 0755)
						req.NoError(err)
						err = mockFs.WriteFile(path.Join("fake-tmp", "README.md"), []byte("its the readme"), 0666)
						req.NoError(err)

						err = mockFs.WriteFile(path.Join("fake-tmp", "Chart.yaml"), []byte(`
---
version: 0.1.0
name: i know what the x5 is
icon: https://kfbr.392/x5.png
`), 0666)
						req.NoError(err)
						return "helm", "fake-tmp", nil
					}).After(inOrder)
				inOrder = mockUi.EXPECT().Info("Detected application type helm").After(inOrder)
				inOrder = mockState.EXPECT().SerializeUpstream("github.com/helm/charts/stable/x5").After(inOrder)
				mockReleaseNotesFetcher.EXPECT().
					ResolveReleaseNotes(ctx, "github.com/helm/charts/stable/x5").
					Return("some release notes", nil)
				inOrder = mockState.EXPECT().SerializeContentSHA("abcdef1234567890").After(inOrder)
				inOrder = mockState.EXPECT().SerializeShipMetadata(api.ShipAppMetadata{
					Version:      "0.1.0",
					Name:         "i know what the x5 is",
					Icon:         "https://kfbr.392/x5.png",
					Readme:       "its the readme",
					ReleaseNotes: "some release notes",
					ContentSHA:   "abcdef1234567890",
					URL:          "github.com/helm/charts/stable/x5",
				}, "helm").After(inOrder)
				inOrder = mockUi.EXPECT().Info("Looking for ship.yaml ...").After(inOrder)
				inOrder = mockUi.EXPECT().Info("ship.yaml not found in upstream, generating default lifecycle for application ...").After(inOrder)
				mockState.EXPECT().SerializeReleaseName("i-know-what-the-x5-is").After(inOrder)

			},
			expectRelease: &api.Release{
				Spec: viperResolver.DefaultHelmRelease("fake-tmp"),
				Metadata: api.ReleaseMetadata{
					ShipAppMetadata: api.ShipAppMetadata{
						Version:      "0.1.0",
						URL:          "github.com/helm/charts/stable/x5",
						Readme:       "its the readme",
						Icon:         "https://kfbr.392/x5.png",
						Name:         "i know what the x5 is",
						ContentSHA:   "abcdef1234567890",
						ReleaseNotes: "some release notes",
					},
				},
			},
		},
		{
			name:     "replicated.app",
			upstream: "replicated.app?customer_id=12345&installation_id=67890",
			expect: func(
				t *testing.T,
				mockUi *ui.MockUi,
				appType *apptype.MockInspector,
				mockState *state.MockManager,
				mockFs afero.Afero,
				mockAppResolver *replicatedapp.MockResolver,
				mockReleaseNotesFetcher *githubclient.MockGitHubReleaseNotesFetcher,
			) {
				inOrder := mockUi.EXPECT().Info("Reading replicated.app?customer_id=12345&installation_id=67890 ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "replicated.app?customer_id=12345&installation_id=67890").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						return "replicated.app", "", nil
					}).After(inOrder)

				inOrder = mockUi.EXPECT().Info("Detected application type replicated.app").After(inOrder)
				inOrder = mockState.EXPECT().SerializeUpstream("replicated.app?customer_id=12345&installation_id=67890").After(inOrder)
				mockAppResolver.EXPECT().ResolveAppRelease(ctx, &replicatedapp2.Selector{
					CustomerID:     "12345",
					InstallationID: "67890",
				}).Return(&api.Release{
					Metadata: api.ReleaseMetadata{
						ChannelName: "appgraph-coolci",
					},
				}, nil).After(inOrder)
			},
			expectRelease: &api.Release{
				Metadata: api.ReleaseMetadata{
					ChannelName: "appgraph-coolci",
				},
			},
		},
		{
			name:     "plain k8s app",
			upstream: "github.com/replicatedhq/test-charts/plain-k8s",
			shaSummer: func(resolver *Resolver, s string) (string, error) {
				return "abcdef1234567890", nil
			},
			expect: func(
				t *testing.T,
				mockUi *ui.MockUi,
				appType *apptype.MockInspector,
				mockState *state.MockManager,
				mockFs afero.Afero,
				mockAppResolver *replicatedapp.MockResolver,
				mockReleaseNotesFetcher *githubclient.MockGitHubReleaseNotesFetcher,
			) {
				req := require.New(t)
				inOrder := mockUi.EXPECT().Info("Reading github.com/replicatedhq/test-charts/plain-k8s ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "github.com/replicatedhq/test-charts/plain-k8s").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						err := mockFs.MkdirAll("fake-tmp", 0755)
						req.NoError(err)
						err = mockFs.WriteFile(path.Join("fake-tmp", "README.md"), []byte("its the readme"), 0644)
						req.NoError(err)
						return "k8s", "fake-tmp", nil
					}).After(inOrder)
				inOrder = mockUi.EXPECT().Info("Detected application type k8s").After(inOrder)
				inOrder = mockState.EXPECT().SerializeUpstream("github.com/replicatedhq/test-charts/plain-k8s").After(inOrder)
				inOrder = mockReleaseNotesFetcher.EXPECT().
					ResolveReleaseNotes(ctx, "github.com/replicatedhq/test-charts/plain-k8s").
					Return("plain-k8s example", nil).After(inOrder)
				inOrder = mockState.EXPECT().SerializeContentSHA("abcdef1234567890").After(inOrder)
				inOrder = mockUi.EXPECT().Info("Looking for ship.yaml ...").After(inOrder)
				inOrder = mockUi.EXPECT().Info("ship.yaml not found in upstream, generating default lifecycle for application ...").After(inOrder)
				mockState.EXPECT().SerializeReleaseName("ship").After(inOrder)
			},
			expectRelease: &api.Release{
				Spec: viperResolver.DefaultRawRelease("base"),
				Metadata: api.ReleaseMetadata{
					ShipAppMetadata: api.ShipAppMetadata{
						URL:          "github.com/replicatedhq/test-charts/plain-k8s",
						Readme:       "its the readme",
						ReleaseNotes: "plain-k8s example",
						ContentSHA:   "abcdef1234567890",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			mockUI := ui.NewMockUi(mc)
			appType := apptype.NewMockInspector(mc)
			mockState := state.NewMockManager(mc)
			mockAppResolver := replicatedapp.NewMockResolver(mc)
			mockReleaseNotesFetcher := githubclient.NewMockGitHubReleaseNotesFetcher(mc)

			// need a real FS because afero.Rename on a memMapFs doesn't copy directories recursively
			fs := afero.Afero{Fs: afero.NewOsFs()}
			tmpdir, err := fs.TempDir("./", test.name)
			req.NoError(err)
			defer fs.RemoveAll(tmpdir)

			mockFs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), tmpdir)}
			// its chrooted to a temp dir, but this needs to exist
			err = mockFs.MkdirAll(".ship/tmp/", 0755)
			req.NoError(err)

			resolver := &Resolver{
				Logger:                    log.NewNopLogger(),
				StateManager:              mockState,
				FS:                        mockFs,
				AppResolver:               mockAppResolver,
				Viper:                     viper.New(),
				ui:                        mockUI,
				appTypeInspector:          appType,
				shaSummer:                 test.shaSummer,
				GitHubReleaseNotesFetcher: mockReleaseNotesFetcher,
			}
			test.expect(t, mockUI, appType, mockState, mockFs, mockAppResolver, mockReleaseNotesFetcher)

			func() {
				defer mc.Finish()
				release, err := resolver.ResolveRelease(ctx, test.upstream)
				req.NoError(err)
				req.Equal(test.expectRelease, release)

			}()
		})
	}
}

func TestResolver_ReadContentSHAForWatch(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		upstream  string
		shaSummer shaSummer
		expect    func(
			t *testing.T,
			appType *apptype.MockInspector,
			mockAppResolver *replicatedapp.MockResolver,
		)
		expectSHA string
	}{
		{
			name:     "happy path replicated.app",
			upstream: "replicated.app/some-tool?customer_id=foo&installation_id=bar",
			expect: func(t *testing.T, appType *apptype.MockInspector, mockAppResolver *replicatedapp.MockResolver) {
				appType.EXPECT().
					DetermineApplicationType(ctx, "replicated.app/some-tool?customer_id=foo&installation_id=bar").
					Return("replicated.app", "fake", nil)
				mockAppResolver.EXPECT().FetchRelease(ctx, &replicatedapp2.Selector{
					CustomerID:     "foo",
					InstallationID: "bar",
				}).Return(&replicatedapp2.ShipRelease{Spec: "its fake"}, nil)
			},
			expectSHA: "a9274e43955abe372d508864d19aa8be39872a39f44c8c5e2e04a4ef98c4aa04", // sha256.Sum256([]byte("its fake"))
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			inspector := apptype.NewMockInspector(mc)
			resolver := replicatedapp.NewMockResolver(mc)

			test.expect(t, inspector, resolver)

			r := &Resolver{
				Logger:           &logger.TestLogger{T: t},
				appTypeInspector: inspector,
				AppResolver:      resolver,
				shaSummer:        test.shaSummer,
				FS:               afero.Afero{Fs: afero.NewMemMapFs()},
			}

			sha, err := r.ReadContentSHAForWatch(ctx, test.upstream)
			req.NoError(err)
			req.Equal(test.expectSHA, sha)
		})
	}
}

func TestResolver_recursiveCopy(t *testing.T) {
	type fileStruct struct {
		name string
		data string
	}

	tests := []struct {
		name        string
		fromPath    string
		destPath    string
		wantErr     bool
		inputFiles  []fileStruct
		outputFiles []fileStruct
	}{
		{
			name:     "one file",
			fromPath: "/test",
			destPath: "/dest",
			wantErr:  false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
				{
					name: "/dest/main.yml",
					data: `filedata`,
				},
			},
		},
		{
			name:     "two files, nested dirs",
			fromPath: "/test",
			destPath: "/dest",
			wantErr:  false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
				{
					name: "/test/a/test/dir/file.txt",
					data: `nested`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
				{
					name: "/test/a/test/dir/file.txt",
					data: `nested`,
				},
				{
					name: "/dest/main.yml",
					data: `filedata`,
				},
				{
					name: "/dest/a/test/dir/file.txt",
					data: `nested`,
				},
			},
		},
		{
			name:     "src does not exist, other files undisturbed",
			fromPath: "/src",
			destPath: "/dest",
			wantErr:  false,
			inputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
			},
			outputFiles: []fileStruct{
				{
					name: "/test/main.yml",
					data: `filedata`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// setup input FS
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			req.NoError(mockFs.MkdirAll(tt.fromPath, os.FileMode(0644)))
			for _, inFile := range tt.inputFiles {
				req.NoError(mockFs.MkdirAll(filepath.Dir(inFile.name), os.FileMode(0644)))
				req.NoError(mockFs.WriteFile(inFile.name, []byte(inFile.data), os.FileMode(0644)))
			}

			r := Resolver{
				FS:     mockFs,
				Logger: log.NewNopLogger(),
			}

			// run copy function
			if err := r.recursiveCopy(tt.fromPath, tt.destPath); (err != nil) != tt.wantErr {
				t.Errorf("Resolver.recursiveCopy() error = %v, wantErr %v", err, tt.wantErr)
			}

			// compare output FS
			// this doesn't validate that there aren't extra files produced, but that shouldn't be a problem
			for _, outFile := range tt.outputFiles {
				fileBytes, err := mockFs.ReadFile(outFile.name)
				req.NoError(err, "reading output file %s", outFile.name)

				req.Equal(outFile.data, string(fileBytes), "compare file %s", outFile.name)
			}
		})
	}
}
