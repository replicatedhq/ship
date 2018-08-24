package planner

import (
	"context"
)

// Execute executes the plan
func (p *CLIPlanner) Execute(ctx context.Context, plan Plan) error {
	for _, step := range plan {
		if err := step.Execute(ctx); err != nil {
			return err
		}
	}
	return nil
}
