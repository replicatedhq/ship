package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type SaveOpts struct {
	PullUrl   string
	SaveUrl   string
	IsPrivate bool
	Filename  string
	Username  string
	Password  string
	Logger    log.Logger
}

func SaveImage(ctx context.Context, saveOpts SaveOpts) chan interface{} {
	ch := make(chan interface{})
	go func() {
		defer close(ch)
		if err := saveImage(ctx, saveOpts, ch); err != nil {
			ch <- err
		}
	}()
	return ch
}

func saveImage(ctx context.Context, saveOpts SaveOpts, progressCh chan interface{}) error {
	debug := level.Debug(log.With(saveOpts.Logger, "method", "saveImage", "image", saveOpts.SaveUrl))

	authOpts := types.AuthConfig{}
	if saveOpts.IsPrivate {
		authOpts.Username = saveOpts.Username
		authOpts.Password = saveOpts.Password
	}

	debug.Log("stage", "create.client")

	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrapf(err, "create docker client")
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
	progressReader, err := cli.ImagePull(ctx, saveOpts.PullUrl, pullOpts)
	if err != nil {
		return errors.Wrapf(err, "pull image %s", saveOpts.PullUrl)
	}
	copyDockerProgress(progressReader, progressCh)

	if saveOpts.PullUrl != saveOpts.SaveUrl {
		debug.Log("stage", "tag", "old.tag", saveOpts.PullUrl, "new.tag", saveOpts.SaveUrl)
		err := cli.ImageTag(ctx, saveOpts.PullUrl, saveOpts.SaveUrl)
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

	imageReader, err := cli.ImageSave(ctx, []string{saveOpts.SaveUrl})
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
