package github

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// Renderer is something that can render a helm asset as part of a planner.Plan
type Renderer interface {
	Execute(
		rootFs root.Fs,
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
	Viper          *viper.Viper
}

func NewRenderer(
	logger log.Logger,
	fs afero.Afero,
	viper *viper.Viper,
	builderBuilder *templates.BuilderBuilder,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		Fs:             fs,
		Viper:          viper,
		BuilderBuilder: builderBuilder,
	}
}

// refactored from planner.plan but I neeeeed tests
func (r *LocalRenderer) Execute(
	rootFs root.Fs,
	asset api.GitHubAsset,
	configGroups []libyaml.ConfigGroup,
	meta api.ReleaseMetadata,
	templateContext map[string]interface{},
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github", "dest", asset.Dest, "description", asset.Description))

		debug.Log("event", "execute")
		basePath := filepath.Dir(asset.Dest)
		debug.Log("event", "mkdirall.attempt", "root", rootFs.RootPath, "dest", asset.Dest, "basePath", basePath)
		if err := rootFs.MkdirAll(basePath, 0755); err != nil {
			debug.Log("event", "mkdirall.fail", "err", err, "root", rootFs.RootPath, "dest", asset.Dest, "basePath", basePath)
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
			return errors.New("github asset returned no files")
		}

		builder, err := r.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "init builder")
		}

		for _, file := range files {
			data, err := base64.StdEncoding.DecodeString(file.Data)
			if err != nil {
				return errors.Wrapf(err, "decode %s", file.Path)
			}

			built, err := builder.String(string(data))
			if err != nil {
				return errors.Wrapf(err, "building %s", file.Path)
			}

			filePath, err := getDestPath(file.Path, asset, builder)
			if err != nil {
				return errors.Wrapf(err, "determining destination for %s", file.Path)
			}

			basePath := filepath.Dir(filePath)
			debug.Log("event", "mkdirall.attempt", "root", rootFs.RootPath, "dest", filePath, "basePath", basePath)
			if err := rootFs.MkdirAll(basePath, 0755); err != nil {
				debug.Log("event", "mkdirall.fail", "err", err, "root", rootFs.RootPath, "dest", filePath, "basePath", basePath)
				return errors.Wrapf(err, "write directory to %s", filePath)
			}

			mode := os.FileMode(0644) // TODO: how to get mode info from github?
			if asset.AssetShared.Mode != os.FileMode(0000) {
				mode = asset.AssetShared.Mode
			}
			if err := rootFs.WriteFile(filePath, []byte(built), mode); err != nil {
				debug.Log("event", "execute.fail", "err", err)
				return errors.Wrapf(err, "Write inline asset to %s", filePath)
			}

			// TODO: check file sha
		}
		return nil
	}
}

func getDestPath(githubPath string, asset api.GitHubAsset, builder *templates.Builder) (string, error) {
	stripPath, err := builder.Bool(asset.StripPath, false)
	if err != nil {
		return "", errors.Wrapf(err, "parse boolean from %q", asset.StripPath)
	}

	destDir, err := builder.String(asset.Dest)
	if err != nil {
		return "", errors.Wrapf(err, "get destination directory from %q", asset.Dest)
	}

	if stripPath {
		// remove asset.Path's directory from the beginning of githubPath
		sourcePathDir := filepath.ToSlash(filepath.Dir(asset.Path)) + "/"
		githubPath = strings.TrimPrefix(githubPath, sourcePathDir)

		// handle cases where the source path was a dir but a trailing slash was not included
		if !strings.HasSuffix(asset.Path, "/") {
			sourcePathBase := filepath.Base(asset.Path) + "/"
			githubPath = strings.TrimPrefix(githubPath, sourcePathBase)
		}
	}

	return filepath.Join(destDir, githubPath), nil
}
