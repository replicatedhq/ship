package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"strings"

	"time"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/lifecycle"
	planner2 "github.com/replicatedhq/ship/pkg/test-mocks/planner"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/stretchr/testify/require"
)

type completestepTestCase struct {
	Name           string
	Lifecycle      []api.Step
	POST           string
	ExpectStatus   int
	ExpectBody     map[string]interface{}
	State          *state2.Lifeycle
	ExpectState    *matchers.Is
	OnExecute      func(d *NavcycleRoutes, step api.Step) error
	WaitForCleanup func() <-chan time.Time

	// gonna move this to another test
	//ExpectLifecycleCalls func(
	//	release *api.Release,
	//	m *lifecycle.MockMessenger,
	//	r *lifecycle.MockRenderer,
	//	d *daemon.MockDaemon, // this really only needs to be a StatusReceiver but I'm too lazy to mockgen one
	//	p *planner2.MockPlanner,
	//)
	//ExpectStepStatuses []struct {
	//	channel  chan interface{}
	//	GET      string
	//	progress interface{}
	//}

}

func TestV2CompleteStep(t *testing.T) {
	tests := []completestepTestCase{
		{
			Name:         "empty",
			Lifecycle:    []api.Step{},
			POST:         "/api/v1/navcycle/step/foo",
			ExpectStatus: 404,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"notFound": map[string]interface{}{},
				},
				"phase": "notFound",
			},
		},
		{
			Name: "complete missing message",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						Contents: "lol",
						StepShared: api.StepShared{
							ID: "foo",
						},
					},
				},
			},
			POST:         "/api/v1/navcycle/step/bar",
			ExpectStatus: 404,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"notFound": map[string]interface{}{},
				},
				"phase": "notFound",
			},
		},
		{
			Name: "complete message",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						Contents: "lol",
						StepShared: api.StepShared{
							ID: "foo",
						},
					},
				},
			},
			POST:         "/api/v1/navcycle/step/foo",
			ExpectStatus: 200,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"message": map[string]interface{}{
						"contents": "lol", "trusted_html": true,
					},
				},
				"phase": "message",
				"actions": []interface{}{
					map[string]interface{}{
						"buttonType":  "primary",
						"text":        "Confirm",
						"loadingText": "Confirming",
						"onclick": map[string]interface{}{
							"uri":    "/navcycle/step/foo",
							"method": "POST",
							"body":   "",
						},
					},
				},

				"progress": map[string]interface{}{
					"source": "v2router", "type": "string", "level": "info", "detail": "success",
				},
			},
			ExpectState: &matchers.Is{
				Describe: "saved state has step foo completed",
				Test: func(v interface{}) bool {
					if versioned, ok := v.(state2.VersionedState); ok {
						_, ok := versioned.V1.Lifecycle.StepsCompleted["foo"]
						return ok
					}
					return false
				},
			},
		},
		{
			Name: "can't complete step with unsatisfied requirement",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						Contents: "spam step",
						StepShared: api.StepShared{
							ID: "spam",
						},
					},
				},
				{
					Message: &api.Message{
						Contents: "lol",
						StepShared: api.StepShared{
							ID:       "foo",
							Requires: []string{"spam"},
						},
					},
				},
			},
			POST:         "/api/v1/navcycle/step/foo",
			ExpectStatus: 400,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"requirementNotMet": map[string]interface{}{
						"required": "spam",
					},
				},
				"phase": "requirementNotMet",
			},
		},
		{
			Name: "fast render completes synchronously",
			Lifecycle: []api.Step{
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID: "make-the-things",
						},
					},
				},
			},
			POST:         "/api/v1/navcycle/step/make-the-things",
			ExpectStatus: 200,
			ExpectState: &matchers.Is{
				Describe: "saved state has step make-the-things completed",
				Test: func(v interface{}) bool {
					if versioned, ok := v.(state2.VersionedState); ok {
						_, ok := versioned.V1.Lifecycle.StepsCompleted["make-the-things"]
						return ok
					}
					return false
				},
			},
			OnExecute: func(d *NavcycleRoutes, step api.Step) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"render": map[string]interface{}{},
				},
				"phase": "render",
				"progress": map[string]interface{}{
					"source": "v2router",
					"type":   "string",
					"level":  "info",
					"detail": "success",
				},
			},
		},
		{
			Name: "slow render (600ms) completes async, within 150ms of api route returning",
			Lifecycle: []api.Step{
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID: "make-the-things",
						},
					},
				},
			},
			POST: "/api/v1/navcycle/step/make-the-things",
			// need to wait until the async task completes before we check all the expected mock calls,
			// otherwise the state won't have been saved yet
			WaitForCleanup: func() <-chan time.Time { return time.After(150 * time.Millisecond) },
			OnExecute: func(d *NavcycleRoutes, step api.Step) error {
				d.StepProgress.Store("make-the-things", daemontypes.StringProgress("unittest", "workin on it"))
				time.Sleep(600 * time.Millisecond)
				return nil
			},
			ExpectStatus: 200,
			ExpectState: &matchers.Is{
				Describe: "saved state has step make-the-things completed",
				Test: func(v interface{}) bool {
					if versioned, ok := v.(state2.VersionedState); ok {
						_, ok := versioned.V1.Lifecycle.StepsCompleted["make-the-things"]
						return ok
					}
					return false
				},
			},
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"render": map[string]interface{}{},
				},
				"phase": "render",
				"progress": map[string]interface{}{
					"source": "unittest",
					"level":  "info",
					"type":   "string",
					"detail": "workin on it",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			release := &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: test.Lifecycle,
					},
				},
			}
			mc := gomock.NewController(t)
			fakeState := state.NewMockManager(mc)
			testLogger := &logger.TestLogger{T: t}
			messenger := lifecycle.NewMockMessenger(mc)
			renderer := lifecycle.NewMockRenderer(mc)
			mockPlanner := planner2.NewMockPlanner(mc)
			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StateManager: fakeState,
				Messenger:    messenger,
				Renderer:     renderer,
				Planner:      mockPlanner,
				StepExecutor: func(d *NavcycleRoutes, step api.Step) error {
					return nil
				},
				StepProgress: &daemontypes.ProgressMap{},
			}

			fakeState.EXPECT().TryLoad().Return(state2.VersionedState{
				V1: &state2.V1{
					Lifecycle: test.State,
				},
			}, nil).AnyTimes()

			if test.ExpectState != nil && test.ExpectState.Test != nil {
				fakeState.EXPECT().Save(test.ExpectState).Return(nil)
			}

			if test.OnExecute != nil {
				v2.StepExecutor = test.OnExecute
			}

			func() {
				_, port, cancelFunc, err := initTestDaemon(t, release, v2)
				defer func() {
					if test.WaitForCleanup != nil {
						<-test.WaitForCleanup()
					}
					mc.Finish()
					cancelFunc()
				}()
				req.NoError(err)
				addr := fmt.Sprintf("http://localhost:%d", port)
				req := require.New(t)

				// send request
				resp, err := http.Post(fmt.Sprintf("%s%s", addr, test.POST), "application/json", strings.NewReader(""))
				req.NoError(err)
				req.Equal(test.ExpectStatus, resp.StatusCode)
				bytes, err := ioutil.ReadAll(resp.Body)
				req.NoError(err)
				var deserializeTarget map[string]interface{}
				err = json.Unmarshal(bytes, &deserializeTarget)
				req.NoError(err)

				diff := deep.Equal(test.ExpectBody, deserializeTarget)
				bodyForDebug, err := json.Marshal(test.ExpectBody)
				if err != nil {
					bodyForDebug = []byte(err.Error())
				}
				req.Empty(diff, "\nexpect: %s\nactual: %s\ndiff: %s", bodyForDebug, string(bytes), strings.Join(diff, "\n"))

			}()
		})
	}
}
