package planner

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render/config"
	"github.com/replicatedcom/ship/pkg/templates"

	"github.com/replicatedhq/libyaml"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

func (p *CLIPlanner) githubStep(asset *api.GithubAsset, configGroups []libyaml.ConfigGroup, meta api.ReleaseMetadata, templateContext map[string]interface{}) Step {
	debug := level.Debug(log.With(p.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "docker", "dest", asset.Dest, "description", asset.Description))
	return Step{
		Dest:        asset.Dest,
		Description: asset.Description,
		Execute: func(ctx context.Context) error {
			debug.Log("event", "execute")
			basePath := filepath.Dir(asset.Dest)
			debug.Log("event", "mkdirall.attempt", "dest", asset.Dest, "basePath", basePath)
			if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", asset.Dest, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", asset.Dest)
			}

			var files []api.GithubFile
			for _, c := range meta.GithubContents {
				if c.Repo == asset.Repo && c.Path == asset.Path && c.Ref == asset.Ref {
					files = c.Files
					break
				}
			}

			if len(files) == 0 {
				level.Info(p.Logger).Log("msg", "no files for asset", "repo", asset.Repo, "path", asset.Path)
				return nil
			}

			configCtx, err := config.NewConfigContext(
				p.Viper, p.Logger,
				configGroups, templateContext)
			if err != nil {
				return errors.Wrap(err, "getting config context")
			}

			builder := templates.NewBuilder(
				templates.NewStaticContext(),
				configCtx,
			)

			for _, file := range files {
				data, err := base64.StdEncoding.DecodeString(file.Data)
				if err != nil {
					return errors.Wrapf(err, "decode %s", file.Path)
				}

				built, err := builder.String(string(data))
				if err != nil {
					return errors.Wrapf(err, "building %s", file.Path)
				}

				filePath := filepath.Join(asset.Dest, file.Path)

				basePath := filepath.Dir(filePath)
				debug.Log("event", "mkdirall.attempt", "dest", filePath, "basePath", basePath)
				if err := p.Fs.MkdirAll(basePath, 0755); err != nil {
					debug.Log("event", "mkdirall.fail", "err", err, "dest", filePath, "basePath", basePath)
					return errors.Wrapf(err, "write directory to %s", filePath)
				}

				mode := os.FileMode(0644) // TODO: how to get mode info from github?
				if err := p.Fs.WriteFile(filePath, []byte(built), mode); err != nil {
					debug.Log("event", "execute.fail", "err", err)
					return errors.Wrapf(err, "Write inline asset to %s", filePath)
				}

				// TODO: check file sha
			}

			return nil
		},
	}
}
