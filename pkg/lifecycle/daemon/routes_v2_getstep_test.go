package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	state2 "github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
)

type getstepTestCase struct {
	Name         string
	Lifecycle    []api.Step
	GET          string
	ExpectStatus int
	ExpectBody   map[string]interface{}
	StepProgress map[string]daemontypes.Progress
	State        *state2.Lifeycle
}

func TestV2GetStep(t *testing.T) {
	tests := []getstepTestCase{
		{
			Name:         "empty",
			Lifecycle:    []api.Step{},
			GET:          "/api/v2/lifecycle/step/foo",
			ExpectStatus: 404,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"notFound": map[string]interface{}{},
				},
				"phase": "notFound",
			},
		},
		{
			Name: "matching step",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "foo",
						},
						Contents: "hi",
					},
				},
			},
			GET:          "/api/v2/lifecycle/step/foo",
			ExpectStatus: 200,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"message": map[string]interface{}{
						"contents":     "hi",
						"trusted_html": true,
					},
				},
				"phase": "message",
				"actions": []interface{}{
					map[string]interface{}{
						"buttonType":  "primary",
						"text":        "Confirm",
						"loadingText": "Confirming",
						"onclick": map[string]interface{}{
							"uri":    "/api/v2/lifecycle/step/foo",
							"method": "POST",
							"body":   "",
						},
					},
				},
			},
		},
		{
			Name: "cant skip steps",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "foo",
						},
						Contents: "hi",
					},
				},
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "bar",
							Requires: []string{
								"foo",
							},
						},
						Contents: "hi",
					},
				},
			},
			GET:          "/api/v2/lifecycle/step/bar",
			ExpectStatus: 400,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"requirementNotMet": map[string]interface{}{
						"required": "foo",
					},
				},
				"phase": "requirementNotMet",
			},
		},
		{
			Name: "can reach step 2 if step 1 complete",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "foo",
						},
						Contents: "hi",
					},
				},
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "bar",
							Requires: []string{
								"foo",
							},
						},
						Contents: "hi from bar",
					},
				},
			},
			State: &state2.Lifeycle{
				StepsCompleted: map[string]interface{}{
					"foo": nil,
				},
			},
			GET:          "/api/v2/lifecycle/step/bar",
			ExpectStatus: 200,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"message": map[string]interface{}{
						"contents":     "hi from bar",
						"trusted_html": true,
					},
				},
				"phase": "message",
				"actions": []interface{}{
					map[string]interface{}{
						"buttonType":  "primary",
						"text":        "Confirm",
						"loadingText": "Confirming",
						"onclick": map[string]interface{}{
							"uri":    "/api/v2/lifecycle/step/bar",
							"method": "POST",
							"body":   "",
						},
					},
				},
			},
		},
		{
			Name: "get step returns progress",
			Lifecycle: []api.Step{
				{
					Message: &api.Message{
						StepShared: api.StepShared{
							ID: "foo",
						},
						Contents: "hi",
					},
				},
			},
			StepProgress: map[string]daemontypes.Progress{
				"foo": daemontypes.StringProgress("v2router", "working"),
			},
			GET:          "/api/v2/lifecycle/step/foo",
			ExpectStatus: 200,
			ExpectBody: map[string]interface{}{
				"currentStep": map[string]interface{}{
					"message": map[string]interface{}{
						"contents":     "hi",
						"trusted_html": true,
					},
				},
				"phase": "message",
				"progress": map[string]interface{}{
					"source": "v2router",
					"type":   "string",
					"level":  "info",
					"detail": "working",
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
			progressmap := &daemontypes.ProgressMap{}
			for key, val := range test.StepProgress {
				progressmap.Store(key, val)
			}
			v2 := &V2Routes{
				Logger:       testLogger,
				StateManager: fakeState,
				StepProgress: progressmap,
			}

			fakeState.EXPECT().TryLoad().Return(state2.VersionedState{
				V1: &state2.V1{
					Lifecycle: test.State,
				},
			}, nil).AnyTimes()

			func() {
				//defer mc.Finish()
				_, port, cancelFunc, err := initTestDaemon(t, release, v2)
				defer cancelFunc()
				req.NoError(err)
				addr := fmt.Sprintf("http://localhost:%d", port)
				req := require.New(t)

				// send request
				resp, err := http.Get(fmt.Sprintf("%s%s", addr, test.GET))
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

func TestHydrateActions(t *testing.T) {
	tests := []struct {
		name string
		step daemontypes.Step
		progress map[string]interface{}
		want []daemontypes.Action
	}{
		{
			name: "message",
			step: daemontypes.NewStep(api.Step{
				Message: &api.Message{
					Contents: "hey there",
				},
			}),
			want: []daemontypes.Action{

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

		})
	}
}

