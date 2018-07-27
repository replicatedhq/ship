package images

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/api/types"
)

type MockImageManager struct{}

func (MockImageManager) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	fmt.Println("Image pulled")
	return ioutil.NopCloser(bytes.NewReader([]byte{})), nil
}
func (MockImageManager) ImageTag(ctx context.Context, source, target string) error {
	fmt.Println("Image tagged")
	return nil
}
func (MockImageManager) ImageSave(ctx context.Context, imageIDs []string) (io.ReadCloser, error) {
	fmt.Println("Image saved")
	return ioutil.NopCloser(bytes.NewReader([]byte{})), nil
}
func (MockImageManager) ImagePush(ctx context.Context, image string, options types.ImagePushOptions) (io.ReadCloser, error) {
	fmt.Println("Image pushed")
	return ioutil.NopCloser(bytes.NewReader([]byte{})), nil
}
