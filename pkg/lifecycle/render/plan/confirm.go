package plan

import (
	"fmt"

	"github.com/pkg/errors"
)

// Confirm shows prompts the user to confirm the plan of assets to render
func (p *CLIPlanner) Confirm(plan Plan) (bool, error) {
	p.UI.Output("\nThis command will generate the following resources:\n")
	for _, step := range plan {
		p.UI.Info(fmt.Sprintf("\t%s", step.Dest))
		if step.Description != "" {
			p.UI.Output(fmt.Sprintf("\t%s", step.Description))

		}
	}
	confirmed, err := p.UI.Ask("\n\nIs this ok? [Y/n]:")
	if err != nil {
		return false, errors.Wrap(err, "confirm plan")
	}
	if confirmed != "" && confirmed != "y" {
		return false, nil
	}

	return true, nil
}
