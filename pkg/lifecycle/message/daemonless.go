package message

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship/pkg/api"
)

func NewDaemonlessMessenger(
	logger log.Logger,
) *DaemonlessMessenger {
	return &DaemonlessMessenger{
		Logger: logger,
	}
}

type DaemonlessMessenger struct {
	Logger log.Logger
}

func (d *DaemonlessMessenger) Execute(ctx context.Context, release *api.Release, step *api.Message) error {
	level.Debug(d.Logger).Log("event", "DaemonlessMessenger.nothingToDo")
	return nil
}
