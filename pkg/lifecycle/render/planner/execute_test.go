package planner

import (
	"context"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	Name   string
	Plan   func(p *CLIPlanner) Plan
	Expect map[string]string
}

func TestExecute(t *testing.T) {

	cases := []testcase{
		{
			Name: "one step, one file",
			Plan: func(p *CLIPlanner) Plan {
				return Plan{
					{
						Description: "lol",
						Dest:        "./install.sh",
						Execute: func(ctx context.Context) error {
							return p.Fs.WriteFile("install.sh", []byte("fake"), 0755)
						},
					},
				}
			},
			Expect: map[string]string{
				"install.sh": "fake",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			planner := &CLIPlanner{
				Logger: log.NewNopLogger(),
				UI:     cli.NewMockUi(),
				Fs:     mockFS,
			}

			plan := test.Plan(planner)
			err := planner.Execute(context.Background(), plan)
			req.NoError(err)

			for file, expected := range test.Expect {
				actual, err := mockFS.ReadFile(file)
				req.NoError(err)
				req.Equal(expected, string(actual))
			}
		})
	}
}
