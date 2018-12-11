package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/test-mocks/lifecycle"
	planner2 "github.com/replicatedhq/ship/pkg/test-mocks/planner"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type postTestCase struct {
	POST         string
	ExpectStatus int
	ExpectBody   map[string]interface{}
	ExpectState  *matchers.Is
}

type completestepTestCase struct {
	Name           string
	Lifecycle      []api.Step
	POSTS          []postTestCase
	State          *state2.Lifeycle
	OnExecute      func(d *NavcycleRoutes, step api.Step) error
	WaitForCleanup func() <-chan time.Time
}

func TestV2CompleteStep(t *testing.T) {
	tests := []completestepTestCase{
		{
			Name:      "empty",
			Lifecycle: []api.Step{},
			POSTS: []postTestCase{
				{
					POST:         "/api/v1/navcycle/step/foo",
					ExpectStatus: 404,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"notFound": map[string]interface{}{},
						},
						"phase": "notFound",
					},
				},
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
			POSTS: []postTestCase{
				{
					POST:         "/api/v1/navcycle/step/bar",
					ExpectStatus: 404,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"notFound": map[string]interface{}{},
						},
						"phase": "notFound",
					},
				},
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
			POSTS: []postTestCase{
				{
					POST:         "/api/v1/navcycle/step/foo",
					ExpectStatus: 200,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"message": map[string]interface{}{
								"contents": "lol", "trusted_html": true,
							},
						},
						"phase": "message",
						"progress": map[string]interface{}{
							"source": "v2router",
							"type":   "json",
							"level":  "info",
							"detail": `{"message":"working","status":"working"}`,
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
			},
		},
		{
			Name: "completing step twice invalidates",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						Contents: "lol",
						StepShared: api.StepShared{
							ID:          "foo",
							Invalidates: []string{"bar"},
						},
					},
				},
				{
					Message: &api.Message{
						Contents: "baz",
						StepShared: api.StepShared{
							ID: "bar",
						},
					},
				},
			},
			POSTS: []postTestCase{
				{
					POST:         "/api/v1/navcycle/step/foo",
					ExpectStatus: 200,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"message": map[string]interface{}{
								"contents": "lol", "trusted_html": true,
							},
						},
						"phase": "message",
						"progress": map[string]interface{}{
							"source": "v2router",
							"type":   "json",
							"level":  "info",
							"detail": `{"message":"working","status":"working"}`,
						},
					},
					ExpectState: &matchers.Is{
						Describe: "saved state has step foo completed and bar uncompleted",
						Test: func(v interface{}) bool {
							if versioned, ok := v.(state2.VersionedState); ok {
								_, fooOk := versioned.V1.Lifecycle.StepsCompleted["foo"]
								_, barOk := versioned.V1.Lifecycle.StepsCompleted["bar"]
								return fooOk && !barOk
							}
							return false
						},
					},
				},
				{
					POST:         "/api/v1/navcycle/step/bar",
					ExpectStatus: 200,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"message": map[string]interface{}{
								"contents": "baz", "trusted_html": true,
							},
						},
						"phase": "message",
						"progress": map[string]interface{}{
							"source": "v2router",
							"type":   "json",
							"level":  "info",
							"detail": `{"message":"working","status":"working"}`,
						},
					},
					ExpectState: &matchers.Is{
						Describe: "saved state has step foo and bar completed",
						Test: func(v interface{}) bool {
							if versioned, ok := v.(state2.VersionedState); ok {
								_, fooOk := versioned.V1.Lifecycle.StepsCompleted["foo"]
								_, barOk := versioned.V1.Lifecycle.StepsCompleted["bar"]
								return fooOk && barOk
							}
							return false
						},
					},
				},
				{
					POST:         "/api/v1/navcycle/step/foo",
					ExpectStatus: 200,
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"message": map[string]interface{}{
								"contents": "lol", "trusted_html": true,
							},
						},
						"phase": "message",
						"progress": map[string]interface{}{
							"source": "v2router",
							"type":   "json",
							"level":  "info",
							"detail": `{"message":"working","status":"working"}`,
						},
					},
					ExpectState: &matchers.Is{
						Describe: "saved state has step foo completed and step bar invalidated",
						Test: func(v interface{}) bool {
							if versioned, ok := v.(state2.VersionedState); ok {
								_, fooOk := versioned.V1.Lifecycle.StepsCompleted["foo"]
								_, barOk := versioned.V1.Lifecycle.StepsCompleted["bar"]
								return fooOk && !barOk
							}
							return false
						},
					},
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
			POSTS: []postTestCase{
				{
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
			},
		},
		{
			Name: "render (60ms) completes async, within 15ms of api route returning",
			Lifecycle: []api.Step{
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID: "make-the-things",
						},
					},
				},
			},
			// need to wait until the async task completes before we check all the expected mock calls,
			// otherwise the state won't have been saved yet
			WaitForCleanup: func() <-chan time.Time { return time.After(300 * time.Millisecond) },
			OnExecute: func(d *NavcycleRoutes, step api.Step) error {
				d.StepProgress.Store("make-the-things", daemontypes.StringProgress("unittest", "workin on it"))
				time.Sleep(60 * time.Millisecond)
				return nil
			},
			POSTS: []postTestCase{
				{
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
					ExpectBody: map[string]interface{}{
						"currentStep": map[string]interface{}{
							"render": map[string]interface{}{},
						},
						"phase": "render",
						"progress": map[string]interface{}{
							"source": "unittest",
							"level":  "info",
							"type":   "json",
							"detail": `{"status":"workin on it"}`,
						},
					},
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
				BuilderBuilder: templates.NewBuilderBuilder(testLogger, viper.New(), fakeState),
				Logger:         testLogger,
				StateManager:   fakeState,
				Messenger:      messenger,
				Renderer:       renderer,
				Planner:        mockPlanner,
				StepExecutor: func(d *NavcycleRoutes, step api.Step) error {
					return nil
				},
				StepProgress: &daemontypes.ProgressMap{},
			}

			fakeState.EXPECT().TryLoad().Return(state2.VersionedState{
				V1: &state2.V1{
					Lifecycle: &state2.Lifeycle{
						StepsCompleted: make(map[string]interface{}),
					},
				},
			}, nil).AnyTimes()

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
				for _, testCase := range test.POSTS {
					if testCase.ExpectState != nil && testCase.ExpectState.Test != nil {
						fakeState.EXPECT().Save(testCase.ExpectState).Return(nil)
					}

					resp, err := http.Post(fmt.Sprintf("%s%s", addr, testCase.POST), "application/json", strings.NewReader(""))
					req.NoError(err)
					req.Equal(testCase.ExpectStatus, resp.StatusCode)
					bytes, err := ioutil.ReadAll(resp.Body)
					req.NoError(err)
					var deserializeTarget map[string]interface{}
					err = json.Unmarshal(bytes, &deserializeTarget)
					req.NoError(err)

					diff := deep.Equal(testCase.ExpectBody, deserializeTarget)
					bodyForDebug, err := json.Marshal(testCase.ExpectBody)
					if err != nil {
						bodyForDebug = []byte(err.Error())
					}
					req.Empty(diff, "\nexpect: %s\nactual: %s\ndiff: %s", bodyForDebug, string(bytes), strings.Join(diff, "\n"))
				}
			}()
		})
	}
}
