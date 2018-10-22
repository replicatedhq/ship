// local.go has methods for resolving a local ship.yaml file, and patching in api.Release info
// that would usually be returned by pg.replicated.com
package replicatedapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
)

func (r *resolver) resolveRunbookRelease() (*ShipRelease, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveRunbookRelease"))
	debug.Log("phase", "load-specs", "from", "runbook", "file", r.Runbook)

	specYAML, err := r.FS.ReadFile(r.Runbook)
	if err != nil {
		return nil, errors.Wrapf(err, "read specs from %s", r.Runbook)
	}
	debug.Log("phase", "load-specs", "from", "runbook", "file", r.Runbook, "spec", specYAML)

	if err := r.persistSpec(specYAML); err != nil {
		return nil, errors.Wrapf(err, "serialize last-used YAML to disk")
	}
	debug.Log("phase", "write-yaml", "from", r.Runbook, "write-location", constants.ReleasePath)

	fakeGithubContents, err := r.loadLocalGitHubContents()
	if err != nil {
		return nil, errors.Wrapf(err, "load fake github contents")
	}

	return &ShipRelease{
		Spec:           string(specYAML),
		ChannelName:    r.SetChannelName,
		ChannelIcon:    r.SetChannelIcon,
		Semver:         r.RunbookReleaseSemver,
		GithubContents: fakeGithubContents,
	}, nil
}

func (r *resolver) loadLocalGitHubContents() ([]GithubContent, error) {
	debug := level.Debug(log.With(r.Logger, "method", "loadLocalGitHubContents"))
	var fakeGithubContents []GithubContent
	for _, content := range r.SetGitHubContents {
		debug.Log("event", "githubcontents.set", "received", content)
		split := strings.Split(content, ":")
		if len(split) != 4 {
			return nil, errors.Errorf("set-github-contents %q invalid, expected a REPO:REPO_PATH:REF:LOCAL_PATH", content)
		}
		repo := split[0]
		repoPath := split[1]
		ref := split[2]
		localpath := split[3]

		debug.Log("event", "githubcontents.loadFiles", "localPath", localpath)
		files, err := r.loadLocalGithubFiles(localpath, repoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "set github files")
		}

		fakeGithubContents = append(fakeGithubContents, GithubContent{
			Repo:  repo,
			Path:  repoPath,
			Ref:   ref,
			Files: files,
		})
		debug.Log("event", "githubcontents.set.finished", "received", content)
	}
	return fakeGithubContents, nil
}

func (r *resolver) loadLocalGithubFiles(localpath string, repoPath string) ([]GithubFile, error) {
	debug := level.Debug(log.With(r.Logger, "method", "loadLocalGitHubFiles"))
	var files []GithubFile
	err := r.FS.Walk(localpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "walk %s from %s", info.Name(), path)
		}

		if info.IsDir() {
			return nil
		}

		walkRepoPath := strings.TrimPrefix(path, localpath)
		if !strings.HasPrefix(walkRepoPath, repoPath) {
			return nil
		}

		contents, err := r.FS.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read %s from %s", info.Name(), path)
		}
		debug.Log("event", "githubcontents.loadFile.complete", "path", path, "name", info.Name())

		encodedData := &bytes.Buffer{}
		encoder := base64.NewEncoder(base64.StdEncoding, encodedData)
		defer encoder.Close()
		encoder.Write(contents)
		sha := fmt.Sprintf("%x", sha256.Sum256(contents))
		files = append(files, GithubFile{
			Name: info.Name(),
			Path: walkRepoPath,
			Sha:  sha,
			Size: info.Size(),
			Data: encodedData.String(),
		})
		return nil
	})
	return files, err
}
