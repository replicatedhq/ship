package specs

import (
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
)

func TestPatchIDs(t *testing.T) {
	tests := []struct {
		name string
		in   api.Lifecycle
		want api.Lifecycle
	}{
		{
			name: "empty",
			in: api.Lifecycle{
				V1: []api.Step{},
			},
			want: api.Lifecycle{
				V1: []api.Step{},
			},
		},
		{
			name: "step with id preserves id",
			in: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "intro"},
							Contents:   "hi there",
						},
					},
				},
			},
			want: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "intro"},
							Contents:   "hi there",
						},
					},
				},
			},
		},
		{
			name: "step with no id receives id",
			in: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							Contents: "hi there",
						},
					},
				},
			},
			want: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "message"},
							Contents:   "hi there",
						},
					},
				},
			},
		},
		{
			name: "mixed ids, added id to render",
			in: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "intro"},
							Contents:   "hi there",
						},
					},
					{
						Render: &api.Render{},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "outro"},
							Contents:   "cya",
						},
					},
				},
			},
			want: api.Lifecycle{
				V1: []api.Step{
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "intro"},
							Contents:   "hi there",
						},
					},
					{
						Render: &api.Render{
							StepShared: api.StepShared{ID: "render"},
						},
					},
					{
						Message: &api.Message{
							StepShared: api.StepShared{ID: "outro"},
							Contents:   "cya",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			patcher := &IDPatcher{Logger: &logger.TestLogger{T: t}}
			out := patcher.EnsureAllStepsHaveUniqueIDs(test.in)

			req.Equal(test.want, out)
		})
	}
}
