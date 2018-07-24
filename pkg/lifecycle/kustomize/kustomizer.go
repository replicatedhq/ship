package kustomize

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon"
)

type Kustomizer interface {
	Execute(ctx context.Context, release api.Release, step api.Kustomize) error
}

func NewKustomizer(daemon daemon.Daemon) Kustomizer {
	return &kustomizer{
		Daemon: daemon,
	}

}

// kustomizer will *try* to pull in the Kustomizer libs from kubernetes-sigs/kustomize,
// if not we'll have to fork. for now it just explodes
type kustomizer struct {
	Daemon daemon.Daemon
}

func (l *kustomizer) Execute(ctx context.Context, release api.Release, step api.Kustomize) error {
	panic("I'm not implemented yet!")
}

