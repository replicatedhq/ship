package github

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
		asset api.GitHubAsset,
		configGroups []libyaml.ConfigGroup,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
	) func(ctx context.Context) error
}

var _ Renderer = &LocalRenderer{}

// LocalRenderer pulls github files from pg
type LocalRenderer struct {
	Logger         log.Logger
	Fs             afero.Afero
	BuilderBuilder *templates.BuilderBuilder
}

func NewRenderer(
	logger log.Logger,
	fs afero.Afero,
	builderBuilder *templates.BuilderBuilder,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		Fs:             fs,
		BuilderBuilder: builderBuilder,
	}
}

// refactored from planner.plan but I neeeeed tests
func (r *LocalRenderer) Execute(
	asset api.GitHubAsset,
	configGroups []libyaml.ConfigGroup,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github", "dest", asset.Dest, "description", asset.Description))

		debug.Log("event", "execute")
		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "dest", asset.Dest, "basePath", basePath)
		if err := r.Fs.MkdirAll(basePath, 0755); err != nil {
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
			level.Info(r.Logger).Log("msg", "no files for asset", "repo", asset.Repo, "path", asset.Path)
			return nil
		}

		configCtx, err := templates.NewConfigContext(r.Logger, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "getting config context")
		}

		builder := r.BuilderBuilder.NewBuilder(
			r.BuilderBuilder.NewStaticContext(),
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
			if err := r.Fs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "dest", filePath, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", filePath)
			}

			mode := os.FileMode(0644) // TODO: how to get mode info from github?
			if err := r.Fs.WriteFile(filePath, []byte(built), mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write inline asset to %s", filePath)
			}

			// TODO: check file sha
		}
		return nil
	}
}
