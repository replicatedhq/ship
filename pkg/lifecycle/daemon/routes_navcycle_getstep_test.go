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
	"github.com/spf13/afero"
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
			GET:          "/api/v1/navcycle/step/foo",
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
			GET:          "/api/v1/navcycle/step/foo",
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
							"uri":    "/navcycle/step/foo",
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
			GET:          "/api/v1/navcycle/step/bar",
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
			GET:          "/api/v1/navcycle/step/bar",
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
							"uri":    "/navcycle/step/bar",
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
			GET:          "/api/v1/navcycle/step/foo",
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
					"type":   "json",
					"level":  "info",
					"detail": `{"status":"working"}`,
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
			v2 := &NavcycleRoutes{
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
		name     string
		progress map[string]daemontypes.Progress
		step     daemontypes.Step
		want     []daemontypes.Action
	}{
		{
			name: "message",
			step: daemontypes.NewStep(api.Step{
				Message: &api.Message{
					Contents: "hey there",
					StepShared: api.StepShared{
						ID: "foo",
					},
				},
			}),
			want: []daemontypes.Action{
				{
					ButtonType:  "primary",
					Text:        "Confirm",
					LoadingText: "Confirming",
					OnClick: daemontypes.ActionRequest{
						URI:    "/navcycle/step/foo",
						Method: "POST",
						Body:   "",
					},
				},
			},
		},
		{
			name: "helmintro",
			step: daemontypes.NewStep(api.Step{
				HelmIntro: &api.HelmIntro{
					StepShared: api.StepShared{
						ID: "yo",
					},
				},
			}),
			want: []daemontypes.Action{
				{
					ButtonType:  "primary",
					Text:        "Get started",
					LoadingText: "Confirming",
					OnClick: daemontypes.ActionRequest{
						URI:    "/navcycle/step/yo",
						Method: "POST",
						Body:   "",
					},
				},
			},
		},
		{
			name: "kustomizeIntro",
			step: daemontypes.NewStep(api.Step{
				KustomizeIntro: &api.KustomizeIntro{
					StepShared: api.StepShared{
						ID: "heyo",
					},
				},
			}),
			want: []daemontypes.Action{
				{
					ButtonType:  "primary",
					Text:        "Next",
					LoadingText: "Next",
					OnClick: daemontypes.ActionRequest{
						URI:    "/navcycle/step/heyo",
						Method: "POST",
						Body:   "",
					},
				},
			},
		},
		{
			name: "completed step",
			step: daemontypes.NewStep(api.Step{
				KustomizeIntro: &api.KustomizeIntro{
					StepShared: api.StepShared{
						ID: "hola",
					},
				},
			}),
			progress: map[string]daemontypes.Progress{
				"hola": daemontypes.JSONProgress("v2router", map[string]interface{}{
					"status":  "success",
					"message": "Step completed successfully.",
				}),
			},
			want: []daemontypes.Action{
				{
					ButtonType:  "primary",
					Text:        "Next",
					LoadingText: "Next",
					OnClick: daemontypes.ActionRequest{
						URI:    "/navcycle/step/hola",
						Method: "POST",
						Body:   "",
					},
				},
			},
		},
		{
			name: "in progress step",
			step: daemontypes.NewStep(api.Step{
				KustomizeIntro: &api.KustomizeIntro{
					StepShared: api.StepShared{
						ID: "adios",
					},
				},
			}),
			progress: map[string]daemontypes.Progress{
				"adios": daemontypes.JSONProgress("v2router", map[string]interface{}{
					"status":  "working",
					"message": "working",
				}),
			},
			want: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			testLogger := &logger.TestLogger{T: t}
			progressmap := &daemontypes.ProgressMap{}
			for key, val := range test.progress {
				progressmap.Store(key, val)
			}

			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StepProgress: progressmap,
			}

			actions := v2.getActions(test.step)
			req.Equal(test.want, actions)
		})
	}
}

func TestHydrateStep(t *testing.T) {
	tests := []struct {
		name  string
		step  daemontypes.Step
		state state2.State
		fs    map[string]string
		want  *daemontypes.StepResponse
	}{
		{
			name: "message",
			step: daemontypes.NewStep(api.Step{
				Message: &api.Message{
					Contents: "hey there",
					StepShared: api.StepShared{
						ID: "foo",
					},
				},
			}),
			state: state2.V0{},
			want: &daemontypes.StepResponse{
				CurrentStep: daemontypes.Step{
					Source: api.Step{
						Message: &api.Message{
							Contents: "hey there",
							StepShared: api.StepShared{
								ID: "foo",
							},
						},
					},
					Message: &daemontypes.Message{
						Contents:    "hey there",
						TrustedHTML: true,
					},
				},
				Phase: "message",
				Actions: []daemontypes.Action{
					{
						ButtonType:  "primary",
						Text:        "Confirm",
						LoadingText: "Confirming",
						OnClick: daemontypes.ActionRequest{
							URI:    "/navcycle/step/foo",
							Method: "POST",
							Body:   "",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			progressmap := &daemontypes.ProgressMap{}
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}
			mockState := state.NewMockManager(mc)

			if test.state != nil {
				mockState.EXPECT().TryLoad().Return(test.state, nil)
			}

			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StepProgress: progressmap,
				Fs:           mockFs,
				StateManager: mockState,
			}

			response, err := v2.hydrateStep(test.step)
			req.NoError(err, "hydrate step")
			req.Equal(test.want, response)
		})
	}
}
