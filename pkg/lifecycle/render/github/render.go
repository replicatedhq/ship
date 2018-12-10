package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/render/root"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/gogetter"
	"github.com/replicatedhq/ship/pkg/state"
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

// LocalRenderer pulls proxied github files from pg
// and pulls no proxy github files directly via git or the git client
type LocalRenderer struct {
	Logger         log.Logger
	Fs             afero.Afero
	BuilderBuilder *templates.BuilderBuilder
	Viper          *viper.Viper
	StateManager   state.Manager
}

func NewRenderer(
	logger log.Logger,
	fs afero.Afero,
	viper *viper.Viper,
	builderBuilder *templates.BuilderBuilder,
	stateManager state.Manager,
) Renderer {
	return &LocalRenderer{
		Logger:         logger,
		Fs:             fs,
		Viper:          viper,
		BuilderBuilder: builderBuilder,
		StateManager:   stateManager,
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

		builder, err := r.BuilderBuilder.FullBuilder(meta, configGroups, templateContext)
		if err != nil {
			return errors.Wrap(err, "init builder")
		}

		debug.Log("event", "resolveProxyGithubAssets")
		files := filterGithubContents(meta.GithubContents, asset)
		if len(files) == 0 {
			level.Info(r.Logger).Log("msg", "no proxy files for asset", "repo", asset.Repo, "path", asset.Path)
			r.debugDumpKnownGithubFiles(meta, asset)

			if asset.Source == "public" || !asset.Proxy {
				debug.Log("event", "resolveNoProxyGithubAssets")
				err := r.resolveNoProxyGithubAssets(asset, builder)
				if err != nil {
					return errors.Wrap(err, "resolveNoProxyGithubAssets")
				}
			} else {
				return errors.New("github asset returned no files")
			}
		}

		return r.resolveProxyGithubAssets(asset, builder, rootFs, files)
	}
}

func (r *LocalRenderer) debugDumpKnownGithubFiles(meta api.ReleaseMetadata, asset api.GitHubAsset) {
	debugStr := "["
	for _, content := range meta.GithubContents {
		debugStr += fmt.Sprintf("%s, ", content.String())

	}
	debugStr += "]"

	level.Debug(r.Logger).Log(
		"msg", "github contents",
		"repo", asset.Repo,
		"path", asset.Path,
		"releaseMeta", debugStr,
	)
}

func filterGithubContents(githubContents []api.GithubContent, asset api.GitHubAsset) []api.GithubFile {
	var filtered []api.GithubFile
	for _, c := range githubContents {
		if c.Repo == asset.Repo && strings.Trim(c.Path, "/") == strings.Trim(asset.Path, "/") && c.Ref == asset.Ref {
			filtered = c.Files
			break
		}
	}
	return filtered
}

func (r *LocalRenderer) resolveProxyGithubAssets(asset api.GitHubAsset, builder *templates.Builder, rootFs root.Fs, files []api.GithubFile) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github", "dest", asset.Dest, "description", asset.Description))

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
	}

	return nil
}

func (r *LocalRenderer) resolveNoProxyGithubAssets(asset api.GitHubAsset, builder *templates.Builder) error {
	debug := level.Debug(log.With(r.Logger, "step.type", "render", "render.phase", "execute", "asset.type", "github", "dest", asset.Dest, "description", asset.Description))
	debug.Log("event", "createUpstream")
	upstream := createUpstreamURL(asset)

	var fetcher apptype.FileFetcher
	localFetchPath := filepath.Join(constants.InstallerPrefixPath, constants.GithubAssetSavePath)
	fetcher = githubclient.NewGithubClient(r.Fs, r.Logger)
	if r.Viper.GetBool("prefer-git") {
		var isSingleFile bool
		var subdir string
		upstream, subdir, isSingleFile = gogetter.UntreeGithub(upstream)
		fetcher = &gogetter.GoGetter{Logger: r.Logger, FS: r.Fs, Subdir: subdir, IsSingleFile: isSingleFile}
	}

	debug.Log("event", "getFiles", "upstream", upstream)
	localPath, err := fetcher.GetFiles(context.Background(), upstream, localFetchPath)
	if err != nil {
		return errors.Wrap(err, "get files")
	}

	debug.Log("event", "getDestPath")
	dest, err := getDestPathNoProxy(asset, builder)
	if err != nil {
		return errors.Wrap(err, "get dest path")
	}

	if filepath.Ext(asset.Path) != "" {
		localPath = filepath.Join(localPath, asset.Path)
	}

	exists, err := r.Fs.Exists(filepath.Dir(dest))
	if err != nil {
		return errors.Wrap(err, "dest dir exists")
	}

	if !exists {
		debug.Log("event", "mkdirall", "dir", filepath.Dir(dest))
		if err := r.Fs.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return errors.Wrap(err, "mkdir all dest dir")
		}
	}

	debug.Log("event", "rename", "from", localPath, "dest", dest)
	if err := r.Fs.Rename(localPath, dest); err != nil {
		return errors.Wrap(err, "rename to dest")
	}

	if err := r.Fs.RemoveAll(localFetchPath); err != nil {
		return errors.Wrap(err, "remove tmp github asset")
	}

	return nil
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

func getDestPathNoProxy(asset api.GitHubAsset, builder *templates.Builder) (string, error) {
	assetPath := asset.Path
	stripPath, err := builder.Bool(asset.StripPath, false)
	if err != nil {
		return "", errors.Wrapf(err, "parse boolean from %q", asset.StripPath)
	}

	destDir, err := builder.String(asset.Dest)
	if err != nil {
		return "", errors.Wrapf(err, "get destination directory from %q", asset.Dest)
	}

	if stripPath {
		if filepath.Ext(assetPath) != "" {
			assetPath = filepath.Base(assetPath)
		} else {
			assetPath = ""
		}
	}

	return filepath.Join(constants.InstallerPrefixPath, destDir, assetPath), nil
}

func createUpstreamURL(asset api.GitHubAsset) string {
	var assetType string
	assetBasePath := filepath.Base(asset.Path)
	if filepath.Ext(assetBasePath) != "" {
		assetType = "blob"
	} else {
		assetType = "tree"
	}

	assetRef := "master"
	if asset.Ref != "" {
		assetRef = asset.Ref
	}

	return path.Join("github.com", asset.Repo, assetType, assetRef, asset.Path)
}
