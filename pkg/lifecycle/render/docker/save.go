package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedcom/ship/pkg/logger"
	"github.com/spf13/viper"
)

// ImageSaver saves an image
type ImageSaver interface {
	SaveImage(ctx context.Context, opts SaveOpts) chan interface{}
}

var _ ImageManager = &docker.Client{}

// ImageManager represents a subset of the docker client interface
type ImageManager interface {
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
	ImageTag(ctx context.Context, source, target string) error
	ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error)
}

type SaveOpts struct {
	PullUrl   string
	SaveUrl   string
	IsPrivate bool
	Filename  string
	Username  string
	Password  string
}

var _ ImageSaver = &DockerSaver{}

// DockerSaver implementes ImageSaver via a docker client
type DockerSaver struct {
	Logger log.Logger
	client ImageManager
}

func SaverFromViper(v *viper.Viper) (*DockerSaver, error) {
	client, err := docker.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "initialize docker client")
	}

	return &DockerSaver{
		Logger: logger.FromViper(v),
		client: client,
	}, nil
}

func (s *DockerSaver) SaveImage(ctx context.Context, saveOpts SaveOpts) chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
		if err := s.saveImage(ctx, saveOpts, ch); err != nil {
			ch <- err
		}
	}()
	return ch
}

func (s *DockerSaver) saveImage(ctx context.Context, saveOpts SaveOpts, progressCh chan interface{}) error {
	debug := level.Debug(log.With(s.Logger, "method", "saveImage", "image", saveOpts.SaveUrl))

	authOpts := types.AuthConfig{}
	if saveOpts.IsPrivate {
		authOpts.Username = saveOpts.Username
		authOpts.Password = saveOpts.Password
	}

	debug.Log("stage", "make.auth")

	authString, err := makeAuthValue(authOpts)
	if err != nil {
		return errors.Wrapf(err, "make auth string")
	}

	debug.Log("stage", "pull")

	pullOpts := types.ImagePullOptions{
		RegistryAuth: authString,
	}
	progressReader, err := s.client.ImagePull(ctx, saveOpts.PullUrl, pullOpts)
	if err != nil {
		return errors.Wrapf(err, "pull image %s", saveOpts.PullUrl)
	}
	copyDockerProgress(progressReader, progressCh)

	if saveOpts.PullUrl != saveOpts.SaveUrl {
		debug.Log("stage", "tag", "old.tag", saveOpts.PullUrl, "new.tag", saveOpts.SaveUrl)
		err := s.client.ImageTag(ctx, saveOpts.PullUrl, saveOpts.SaveUrl)
		if err != nil {
			return errors.Wrapf(err, "tag image %s -> %s", saveOpts.PullUrl, saveOpts.SaveUrl)
		}
	}

	debug.Log("stage", "file.create")

	outFile, err := os.Create(saveOpts.Filename)
	if err != nil {
		return errors.Wrapf(err, "open file %s", saveOpts.Filename)
	}
	defer outFile.Close()

	debug.Log("stage", "save")

	imageReader, err := s.client.ImageSave(ctx, []string{saveOpts.SaveUrl})
	if err != nil {
		return errors.Wrapf(err, "save image %s", saveOpts.SaveUrl)
	}
	defer imageReader.Close()

	_, err = io.Copy(outFile, imageReader)
	if err != nil {
		return errors.Wrapf(err, "copy image to file")
	}

	debug.Log("stage", "done")

	return nil
}

func makeAuthValue(authOpts types.AuthConfig) (string, error) {
	jsonified, err := json.Marshal(authOpts)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonified), nil
}
