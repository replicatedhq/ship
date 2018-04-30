package plan

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/spf13/afero"
)

// A Plan is a list of PlanSteps to execute
type Plan []Step

// A Step describes a single unit of work that Ship will do
// to render the application
type Step struct {
	Description string `json:"description" yaml:"description" hcl:"description"`
	Dest        string `json:"dest" yaml:"dest" hcl:"dest"`
	Execute     func(ctx context.Context) error
	Err         error
}

// Planner is a thing that can plan and execute rendering
type Planner interface {
	Build([]api.Asset, api.ReleaseMetadata, map[string]interface{}) Plan
	Confirm(Plan) (bool, error)
	Execute(context.Context, Plan) error
}

// CLIPlanner is the default Planner
type CLIPlanner struct {
	Logger log.Logger
	Fs     afero.Afero
	UI     cli.Ui
}
