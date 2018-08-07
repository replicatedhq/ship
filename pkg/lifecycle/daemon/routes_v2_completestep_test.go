package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"strings"

	"context"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/lifecycle"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/replicatedhq/ship/pkg/testing/matchers"
	"github.com/stretchr/testify/require"
)

type completestepTestCase struct {
	Name                 string
	Lifecycle            []api.Step
	POST                 string
	ExpectStatus         int
	ExpectBody           map[string]interface{}
	State                *state2.Lifeycle
	ExpectState          *matchers.Is
	ExpectLifecycleCalls func(
		release *api.Release,
		m *lifecycle.MockMessenger,
	)
}

func TestV2CompleteStep(t *testing.T) {
	tests := []completestepTestCase{
		{
			Name:         "empty",
			Lifecycle:    []api.Step{},
			POST:         "/api/v2/lifecycle/step/foo",
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
			POST:         "/api/v2/lifecycle/step/bar",
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
			POST:         "/api/v2/lifecycle/step/foo",
			ExpectStatus: 200,
			ExpectBody: map[string]interface{}{
				"status": "success",
			},
			ExpectLifecycleCalls: func(release *api.Release, m *lifecycle.MockMessenger) {
				m.EXPECT().Execute(context.Background(), release, &api.Message{
					Contents: "lol",
					StepShared: api.StepShared{
						ID: "foo",
					},
				})
			},
			ExpectState: &matchers.Is{
				Describe: "saved stated has step foo completed",
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
			POST:         "/api/v2/lifecycle/step/foo",
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
			v2 := &V2Routes{
				Logger:       testLogger,
				StateManager: fakeState,
				Messenger:    messenger,
			}

			fakeState.EXPECT().TryLoad().Return(state2.VersionedState{
				V1: &state2.V1{
					Lifecycle: test.State,
				},
			}, nil).AnyTimes()

			if test.ExpectLifecycleCalls != nil {
				test.ExpectLifecycleCalls(release, messenger)
			}

			if test.ExpectState != nil && test.ExpectState.Test != nil {
				fakeState.EXPECT().Save(test.ExpectState).Return(nil)
			}

			func() {
				defer mc.Finish()
				_, port, cancelFunc, err := initTestDaemon(t, release, v2)
				defer cancelFunc()
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
				req.Empty(diff, "\nexpect: %s\nactual: %s", bodyForDebug, string(bytes))

			}()
		})
	}
}
