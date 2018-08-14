package kustomizeintro

import (
	"context"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle"
)

type kustomizeIntro struct{}

func NewKustomizeIntro() lifecycle.KustomizeIntro {
	return &kustomizeIntro{}
}

func (k *kustomizeIntro) Execute(context.Context, *api.Release, api.KustomizeIntro) error {
	return nil
}
