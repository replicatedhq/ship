package planner

import (
	"testing"

	"github.com/go-kit/kit/log"
	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/test-mocks/ui"
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
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockUI := ui.NewMockUi(mc)
			planner := &CLIPlanner{
				Logger: log.NewNopLogger(),
				UI:     mockUI,
			}

			mockUI.EXPECT().Ask(test.ExpectAsk).Return(test.Answer, test.AnswerErr)
			mockUI.EXPECT().Info(test.ExpectInfo).Return()
			mockUI.EXPECT().Output("\nThis command will generate the following resources:\n").Return()

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
