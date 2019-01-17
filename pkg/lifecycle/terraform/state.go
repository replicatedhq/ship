package terraform

import (
	"bytes"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/hashicorp/terraform/terraform"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
)

type stateSaver func(debug log.Logger, fs afero.Afero, statemanager state.Manager, dir string) error
type stateRestorer func(debug log.Logger, fs afero.Afero, statemanager state.Manager, dir string) error

// this is not on a struct because its used by two and de-duping those the *right* way is
// going to be a big-ish undertaking
func persistState(debug log.Logger, fs afero.Afero, statemanager state.Manager, dir string) error {
	// write terraform state to state.json
	statePath := path.Join(dir, "terraform.tfstate")
	debug.Log("event", "tfstate.readfile", "path", statePath)
	tfstate, err := fs.ReadFile(statePath)
	if err != nil {
		return errors.Wrapf(err, "load state from %s", statePath)
	}

	debug.Log("event", "tfstate.unmarshal", "path", statePath)
	tfstatev3, err := terraform.ReadState(bytes.NewReader(tfstate))
	if err != nil {
		return errors.Wrapf(err, "unmarshal tf state")
	}

	debug.Log("event", "state.load", "path", statePath)
	shipstate, err := statemanager.TryLoad()
	if err != nil {
		return errors.Wrapf(err, "load ship state")
	}
	versioned := shipstate.Versioned()
	if versioned.V1.Terraform == nil {
		versioned.V1.Terraform = &state.Terraform{}
	}
	versioned.V1.Terraform.RawState = string(tfstate)
	versioned.V1.Terraform.State = tfstatev3
	debug.Log("event", "state.save", "path", statePath)
	err = statemanager.Save(versioned)
	if err != nil {
		return errors.Wrapf(err, "save ship state")
	}

	return nil
}

func restoreState(debug log.Logger, fs afero.Afero, statemanager state.Manager, dir string) error {

	debug.Log("event", "state.load")
	shipstate, err := statemanager.TryLoad()
	if err != nil {
		return errors.Wrapf(err, "load ship state")
	}

	versioned := shipstate.Versioned()
	if versioned.V1.Terraform == nil || versioned.V1.Terraform.RawState == "" {
		debug.Log("event", "tfstate.noPreviousState")
		return nil
	}

	statePath := path.Join(dir, "terraform.tfstate")
	debug.Log("event", "tfstate.writeFile", "path", statePath)

	err = fs.WriteFile(statePath, []byte(versioned.V1.Terraform.RawState), 0644)
	if err != nil {
		return errors.Wrapf(err, "write state file")
	}
	debug.Log("event", "tfstate.saved", "path", statePath)
	return nil
}
