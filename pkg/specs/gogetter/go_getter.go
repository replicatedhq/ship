package gogetter

import (
	"context"
	"fmt"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
)

type GoGetter struct {
	Logger log.Logger
	FS     afero.Afero
}

// TODO figure out how to copy files from host into afero filesystem for testing, or how to force go-getter to fetch into afero
func (g *GoGetter) GetFiles(ctx context.Context, upstream, savePath string) error {
	debug := level.Debug(g.Logger)
	debug.Log("event", "gogetter.GetFiles", "upstream", upstream, "savePath", savePath)

	err := getter.GetAny(savePath, upstream)
	if err != nil {
		return errors.Wrap(err, "fetch contents with go-getter")
	}

	// if there is a `.git` directory, remove it - it's dynamic and will break the content hash used by `ship update`
	gitPresent, err := g.FS.Exists(path.Join(savePath, ".git"))
	if err != nil {
		return errors.Wrap(err, "check for .git directory")
	}
	if gitPresent {
		err := g.FS.RemoveAll(path.Join(savePath, ".git"))
		if err != nil {
			return errors.Wrap(err, "remove .git directory")
		}
	}
	debug.Log("event", "gitPresent.check", "gitPresent", gitPresent)

	return nil
}

func IsGoGettable(path string) bool {
	_, err := getter.Detect(path, "", getter.Detectors)
	if err != nil {
		return false
	}
	return true
}

// if this path is a github path of the form `github.com/OWNER/REPO/tree/REF/SUBDIR` or `github.com/OWNER/REPO/SUBDIR`,
// change it to the go-getter form of `github.com/OWNER/REPO?ref=REF//SUBDIR` with a default ref of master
// otherwise return the unmodified path
func UntreeGithub(path string) string {
	githubURL, err := util.ParseGithubURL(path, "master")
	if err != nil {
		return path
	}
	return fmt.Sprintf("github.com/%s/%s?ref=%s//%s", githubURL.Owner, githubURL.Repo, githubURL.Ref, githubURL.Subdir)
}
