package plan

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/gojuno/minimock"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/test-fixtures/ui"
	"github.com/stretchr/testify/require"
)

type confirmTest struct {
	Name         string
	Plan         Plan
	ExpectInfo   string
	ExpectAsk    string
	Answer       string
	AnswerErr    error
	ExpectResult bool
	ExpectError  string
}

func TestConfirm(t *testing.T) {

	cases := []confirmTest{
		{
			Name: "single step plan",
			Plan: Plan{
				{
					Dest: "./install.sh",
				},
			},
			ExpectInfo:   "\t./install.sh",
			ExpectAsk:    "\n\nIs this ok? [Y/n]:",
			Answer:       "",
			ExpectResult: true,
			ExpectError:  "",
		},
		{
			Name: "answer no",
			Plan: Plan{
				{
					Dest: "./install.sh",
				},
			},
			ExpectInfo:   "\t./install.sh",
			ExpectAsk:    "\n\nIs this ok? [Y/n]:",
			Answer:       "n",
			AnswerErr:    nil,
			ExpectResult: false,
			ExpectError:  "",
		},
		{
			Name: "error on ask",
			Plan: Plan{
				{
					Dest: "./install.sh",
				},
			},
			ExpectInfo:   "\t./install.sh",
			ExpectAsk:    "\n\nIs this ok? [Y/n]:",
			Answer:       "",
			AnswerErr:    errors.New("Interrupted"),
			ExpectResult: false,
			ExpectError:  "confirm plan: Interrupted",
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			mc := minimock.NewController(t)
			mockUI := ui.NewUiMock(mc)
			planner := &CLIPlanner{
				Logger: log.NewNopLogger(),
				UI:     mockUI,
			}

			mockUI.AskMock.Expect(test.ExpectAsk).Return(test.Answer, test.AnswerErr)
			mockUI.InfoMock.Expect(test.ExpectInfo).Return()
			mockUI.OutputMock.Expect("\nThis command will generate the following resources:\n").Return()

			result, err := planner.Confirm(test.Plan)

			require.New(t).Equal(test.ExpectResult, result)
			if test.ExpectError == "" {
				require.NoError(t, err)
			} else {
				require.New(t).Equal(test.ExpectError, err.Error())
			}
		})
	}
}
