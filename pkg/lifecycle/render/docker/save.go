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
)

type SaveOpts struct {
	PullUrl   string
	SaveUrl   string
	IsPrivate bool
	Filename  string
	Username  string
	Password  string
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
	authOpts := types.AuthConfig{}
	if saveOpts.IsPrivate {
		authOpts.Username = saveOpts.Username
		authOpts.Password = saveOpts.Password
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrapf(err, "create docker client")
	}

	authString, err := makeAuthValue(authOpts)
	if err != nil {
		return errors.Wrapf(err, "make auth string")
	}

	pullOpts := types.ImagePullOptions{
		RegistryAuth: authString,
	}
	progressReader, err := cli.ImagePull(ctx, saveOpts.PullUrl, pullOpts)
	if err != nil {
		return errors.Wrapf(err, "pull image %s", saveOpts.PullUrl)
	}
	copyDockerProgress(progressReader, progressCh)

	if saveOpts.PullUrl != saveOpts.SaveUrl {
		err := cli.ImageTag(ctx, saveOpts.PullUrl, saveOpts.SaveUrl)
		if err != nil {
			return errors.Wrapf(err, "tag image %s -> %s", saveOpts.PullUrl, saveOpts.SaveUrl)
		}
	}

	outFile, err := os.Create(saveOpts.Filename)
	if err != nil {
		return errors.Wrapf(err, "open file %s", saveOpts.Filename)
	}
	defer outFile.Close()

	imageReader, err := cli.ImageSave(ctx, []string{saveOpts.SaveUrl})
	if err != nil {
		return errors.Wrapf(err, "save image %s", saveOpts.SaveUrl)
	}
	defer imageReader.Close()

	_, err = io.Copy(outFile, imageReader)
	if err != nil {
		return errors.Wrapf(err, "copy image to file")
	}

	return nil
}

func makeAuthValue(authOpts types.AuthConfig) (string, error) {
	jsonified, err := json.Marshal(authOpts)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonified), nil
}
