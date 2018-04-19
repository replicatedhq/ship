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

func SaveImage(ctx context.Context, image string, dstFile string, authOpts types.AuthConfig) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrapf(err, "create docker client")
	}

	authString, err := makeAuthValue(authOpts)
	if err != nil {
		return errors.Wrapf(err, "make auth string")
	}

	opts := types.ImagePullOptions{
		RegistryAuth: authString,
	}
	progressReader, err := cli.ImagePull(ctx, image, opts)
	if err != nil {
		return errors.Wrapf(err, "pull image %s", image)
	}
	io.Copy(os.Stdout, progressReader)

	outFile, err := os.Create(dstFile)
	if err != nil {
		return errors.Wrapf(err, "open file %s", dstFile)
	}
	defer outFile.Close()

	imageReader, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return errors.Wrapf(err, "save image %s", image)
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
