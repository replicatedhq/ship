package planner

import (
	"context"

	"github.com/hashicorp/go-multierror"
)

// Execute executes the plan
func (p *CLIPlanner) Execute(ctx context.Context, plan Plan) error {
	var multiError *multierror.Error

	for _, step := range plan {
		multiError = multierror.Append(multiError, step.Execute(ctx))
	}

	return multiError.ErrorOrNil()
}
