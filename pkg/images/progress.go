package images

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// TODO: This only supports json messages like this one:
// {"status":"Waiting","progressDetail":{},"id":"462d60a56b09"}

type Progress struct {
	ID             string      `json:"id"` // this will be layer ID
	Status         string      `json:"status"`
	ProgressDetail interface{} `json:"progressDetail"`
}

func copyDockerProgress(reader io.ReadCloser, ch chan interface{}) error {
	dec := json.NewDecoder(reader)
	for {
		var m Progress
		if err := dec.Decode(&m); err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "copy docker progress")
		}
		ch <- m
	}
}

// This is a hack, Docker progress does not throw an error if it fails
// to connect to a Docker regustry. Watching for `Preparing` message to know if
// push connection is successful, since EOF is sent in both success and failure
// cases.
func copyDockerProgressPush(reader io.ReadCloser, ch chan interface{}) error {
	dec := json.NewDecoder(reader)
	var preparingToPush bool
	for {
		var m Progress
		if err := dec.Decode(&m); err == io.EOF {
			if !preparingToPush {
				return errors.New("Unable to push Docker image")
			}
			return nil
		} else if err != nil {
			return errors.Wrap(err, "copy docker progress")
		}
		if m.Status == "Preparing" {
			preparingToPush = true
		}
		ch <- m
	}
}
