package docker

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// TODO: This only supports json messages like this one:
// {"status":"Waiting","progressDetail":{},"id":"462d60a56b09"}

type DockerProgress struct {
	ID             string      `json:"id"` // this will be layer ID
	Status         string      `json:"status"`
	ProgressDetail interface{} `json:"progressDetail"`
}

func copyDockerProgress(reader io.ReadCloser, ch chan interface{}) error {
	dec := json.NewDecoder(reader)
	for {
		var m DockerProgress
		if err := dec.Decode(&m); err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "copy docker progress")
		}
		ch <- m
	}
}
