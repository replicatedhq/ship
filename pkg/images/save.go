package images

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"

	"fmt"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
	PullURL   string
	SaveURL   string
	IsPrivate bool
	Filename  string
	Username  string
	Password  string
}

var _ ImageSaver = &CLISaver{}

// CLISaver implementes ImageSaver via a docker client
type CLISaver struct {
	Logger log.Logger
	client ImageManager
}

func NewImageSaver(logger log.Logger, client *docker.Client) ImageSaver {
	return &CLISaver{
		Logger: logger,
		client: client,
	}
}

func (s *CLISaver) SaveImage(ctx context.Context, saveOpts SaveOpts) chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
		if err := s.saveImage(ctx, saveOpts, ch); err != nil {
			ch <- err
		}
	}()
	return ch
}

func (s *CLISaver) saveImage(ctx context.Context, saveOpts SaveOpts, progressCh chan interface{}) error {
	debug := level.Debug(log.With(s.Logger, "method", "saveImage", "image", saveOpts.SaveURL))

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
	progressReader, err := s.client.ImagePull(ctx, saveOpts.PullURL, pullOpts)
	if err != nil {
		return errors.Wrapf(err, "pull image %s", saveOpts.PullURL)
	}
	copyDockerProgress(progressReader, progressCh)

	if saveOpts.PullURL != saveOpts.SaveURL {
		debug.Log("stage", "tag", "old.tag", saveOpts.PullURL, "new.tag", saveOpts.SaveURL)
		err := s.client.ImageTag(ctx, saveOpts.PullURL, saveOpts.SaveURL)
		if err != nil {
			return errors.Wrapf(err, "tag image %s -> %s", saveOpts.PullURL, saveOpts.SaveURL)
		}
	}

	debug.Log("stage", "file.create")

	outFile, err := os.Create(saveOpts.Filename)
	if err != nil {
		return errors.Wrapf(err, "open file %s", saveOpts.Filename)
	}
	defer outFile.Close()

	debug.Log("stage", "save")

	progressCh <- Progress{
		ID:     saveOpts.SaveURL,
		Status: fmt.Sprintf("Saving %s", saveOpts.SaveURL),
	}

	imageReader, err := s.client.ImageSave(ctx, []string{saveOpts.SaveURL})
	if err != nil {
		return errors.Wrapf(err, "save image %s", saveOpts.SaveURL)
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
