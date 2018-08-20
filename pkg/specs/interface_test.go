package specs

import (
	"context"
	"testing"

	"path"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	replicatedapp2 "github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/test-mocks/apptype"
	"github.com/replicatedhq/ship/pkg/test-mocks/replicatedapp"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/ui"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestResolver_ResolveRelease(t *testing.T) {
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
			) {
				req := require.New(t)
				inOrder := mockUi.EXPECT().Info("Reading github.com/helm/charts/stable/x5 ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "github.com/helm/charts/stable/x5").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						err := mockFs.MkdirAll("chart", 0755)
						req.NoError(err)
						err = mockFs.WriteFile(path.Join("chart", "README.md"), []byte("its the readme"), 0666)
						req.NoError(err)

						err = mockFs.WriteFile(path.Join("chart", "Chart.yaml"), []byte(`
---
version: 0.1.0
name: i know what the x5 is
icon: https://kfbr.392/x5.png
`), 0666)
						req.NoError(err)
						return "helm", "chart", nil
					}).After(inOrder)
				inOrder = mockUi.EXPECT().Info("Detected application type helm").After(inOrder)
				inOrder = mockState.EXPECT().SerializeUpstream("github.com/helm/charts/stable/x5").After(inOrder)
				inOrder = mockState.EXPECT().SerializeContentSHA("abcdef1234567890").After(inOrder)
				inOrder = mockUi.EXPECT().Info("Looking for ship.yaml ...").After(inOrder)
				inOrder = mockUi.EXPECT().Info("ship.yaml not found in upstream, generating default lifecycle for application ...").After(inOrder)

			},
			expectRelease: &api.Release{
				Spec: DefaultHelmRelease("chart"),
				Metadata: api.ReleaseMetadata{
					ShipAppMetadata: api.ShipAppMetadata{
						Version:    "0.1.0",
						URL:        "github.com/helm/charts/stable/x5",
						Readme:     "its the readme",
						Icon:       "https://kfbr.392/x5.png",
						Name:       "i know what the x5 is",
						ContentSHA: "abcdef1234567890",
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
			) {
				inOrder := mockUi.EXPECT().Info("Reading replicated.app?customer_id=12345&installation_id=67890 ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "replicated.app?customer_id=12345&installation_id=67890").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						return "replicated.app", "", nil
					}).After(inOrder)

				inOrder = mockUi.EXPECT().Info("Detected application type replicated.app").After(inOrder)
				inOrder = mockAppResolver.EXPECT().ResolveAppRelease(ctx, &replicatedapp2.Selector{
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
			) {
				req := require.New(t)
				inOrder := mockUi.EXPECT().Info("Reading github.com/replicatedhq/test-charts/plain-k8s ...")
				inOrder = mockUi.EXPECT().Info("Determining application type ...").After(inOrder)
				inOrder = appType.EXPECT().
					DetermineApplicationType(ctx, "github.com/replicatedhq/test-charts/plain-k8s").
					DoAndReturn(func(context.Context, string) (string, string, error) {
						err := mockFs.MkdirAll("base", 0755)
						req.NoError(err)
						err = mockFs.WriteFile(path.Join("base", "README.md"), []byte("its the readme"), 0644)
						req.NoError(err)
						return "k8s", "base", nil
					}).After(inOrder)
				inOrder = mockUi.EXPECT().Info("Detected application type k8s").After(inOrder)
				inOrder = mockState.EXPECT().SerializeUpstream("github.com/replicatedhq/test-charts/plain-k8s").After(inOrder)
				inOrder = mockState.EXPECT().SerializeContentSHA("abcdef1234567890").After(inOrder)
				inOrder = mockUi.EXPECT().Info("Looking for ship.yaml ...").After(inOrder)
				inOrder = mockUi.EXPECT().Info("ship.yaml not found in upstream, generating default lifecycle for application ...").After(inOrder)

			},
			expectRelease: &api.Release{
				Spec: DefaultRawRelease("base"),
				Metadata: api.ReleaseMetadata{
					ShipAppMetadata: api.ShipAppMetadata{
						URL:        "github.com/replicatedhq/test-charts/plain-k8s",
						Readme:     "its the readme",
						ContentSHA: "abcdef1234567890",
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
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			resolver := &Resolver{
				Logger:           log.NewNopLogger(),
				StateManager:     mockState,
				FS:               mockFs,
				AppResolver:      mockAppResolver,
				ui:               mockUI,
				appTypeInspector: appType,
				shaSummer:        test.shaSummer,
			}
			test.expect(t, mockUI, appType, mockState, mockFs, mockAppResolver)

			func() {
				defer mc.Finish()
				release, err := resolver.ResolveRelease(ctx, test.upstream)
				req.NoError(err)
				req.Equal(test.expectRelease, release)

			}()
		})
	}
}
