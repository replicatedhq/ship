package kustomize

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
)

func (l *Kustomizer) writePostKustomizeFiles(step api.Kustomize, postKustomizeFiles []util.PostKustomizeFile) error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "writePostKustomizeFiles"))

	return util.WritePostKustomizeFiles(debug, l.FS, step.Dest, postKustomizeFiles)
}

func (l *Kustomizer) maybeCleanupKustomizeState() error {
	debug := level.Debug(log.With(l.Logger, "struct", "daemonless.kustomizer", "method", "maybeCleanupKustomizeState"))

	if l.Viper == nil {
		debug.Log("event", "skip.kustomize.removal.no.viper")
		return nil
	}

	kustomizeInState := l.Viper.GetBool(constants.KustomizeInStateFlag)
	if !kustomizeInState {
		debug.Log("event", "kustomize.state.removal.given.flag")

		currentState, err := l.State.CachedState()
		if err != nil {
			return errors.Wrap(err, "load state")
		}

		currentKustomize := currentState.CurrentKustomize()
		if currentKustomize == nil {
			return nil
		}

		currentOverlay := currentKustomize.Ship()
		currentOverlay.Resources = nil
		currentOverlay.Patches = nil

		currentKustomize.Overlays["ship"] = currentOverlay

		if len(currentOverlay.ExcludedBases) == 0 {
			// remove the kustomize struct entirely
			currentKustomize = nil
		}

		err = l.State.SaveKustomize(currentKustomize)
		if err != nil {
			return errors.Wrap(err, "save removed kustomize state")
		}
	} else {
		debug.Log("event", "skip.kustomize.removal.given.flag")
	}

	return nil
}
