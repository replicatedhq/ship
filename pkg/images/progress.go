package images

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

// TODO: This only supports json messages like this one:
// {"status":"Waiting","progressDetail":{},"id":"462d60a56b09"}

type Progress struct {
	ID             string      `json:"id"` // this will be layer ID
	Status         string      `json:"status"`
	Image          string      `json:"image"`
	ProgressDetail interface{} `json:"progressDetail"`
}

func copyDockerProgress(debug log.Logger, image string, reader io.ReadCloser, ch chan interface{}) error {
	dec := json.NewDecoder(reader)
	for {
		var jm jsonmessage.JSONMessage
		if err := dec.Decode(&jm); err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrap(err, "copy docker progress")
		} else if jm.Error != nil {
			return jm.Error
		}

		debug.Log("event", "docker.JSONMessage.receive", "JSONMessage", fmt.Sprintf("%+v", jm))

		ch <- Progress{
			ID:             jm.ID,
			Status:         jm.Status,
			Image:          image,
			ProgressDetail: jm.Progress,
		}
	}
}
