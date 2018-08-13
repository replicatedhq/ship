package render

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"time"

	_ "github.com/replicatedhq/ship/pkg/lifecycle/render/test-cases"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/planner"
	"github.com/replicatedhq/ship/pkg/state"
	mockdaemon "github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	mockplanner "github.com/replicatedhq/ship/pkg/test-mocks/planner"
	state2 "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/ui"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestRenderNoConfig(t *testing.T) {
	ctx := context.Background()

	tests := loadTestCases(t, filepath.Join("test-cases", "render-inline.yaml"))

	for _, test := range tests[:1] {
		t.Run(test.Name, func(t *testing.T) {
			mc := gomock.NewController(t)

			mockUI := ui.NewMockUi(mc)
			p := mockplanner.NewMockPlanner(mc)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			mockState := state2.NewMockManager(mc)
			mockDaemon := mockdaemon.NewMockDaemon(mc)

			renderer := &noconfigrenderer{
				Logger: log.NewNopLogger(),
				Now:    time.Now,
			}
			renderer.Fs = mockFS
			renderer.UI = mockUI
			renderer.Planner = p
			renderer.StateManager = mockState

			prog := mockDaemon.EXPECT().SetProgress(ProgressRead)
			prog = mockDaemon.EXPECT().SetProgress(ProgressRender).After(prog)
			prog = mockDaemon.EXPECT().SetStepName(ctx, daemontypes.StepNameConfirm).After(prog)
			mockDaemon.EXPECT().ClearProgress().After(prog)

			renderer.StatusReceiver = mockDaemon

			release := &api.Release{Spec: test.Spec}

			func() {
				defer mc.Finish()

				mockState.EXPECT().TryLoad().Return(state.V0(test.ViperConfig), nil)

				p.EXPECT().
					Build(test.Spec.Assets.V1, test.Spec.Config.V1, gomock.Any(), test.ViperConfig).
					Return(planner.Plan{}, nil)

				p.EXPECT().
					Execute(ctx, planner.Plan{}).
					Return(nil)

				err := renderer.Execute(ctx, release, &api.Render{})
				assert.NoError(t, err)
			}()
		})
	}
}
