package images

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/url"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"
	mockimages "github.com/replicatedhq/ship/pkg/test-mocks/images"
	logger2 "github.com/replicatedhq/ship/pkg/testing/logger"
)

func Test_buildDestinationParams(t *testing.T) {
	basicURL, _ := url.Parse("docker://registry.somebigbank.com:9800/myregistry/myapi:1")
	urlWithAuth, _ := url.Parse("docker://username:password@registry.somebigbank.com:9800/myregistry/myapi:1")
	type args struct {
		destinationURL *url.URL
	}
	tests := []struct {
		name    string
		args    args
		want    DestinationParams
		wantErr bool
	}{
		{
			name: "Basic URL",
			args: args{
				destinationURL: basicURL,
			},
			want: DestinationParams{
				AuthConfig:           types.AuthConfig{},
				DestinationImageName: "registry.somebigbank.com:9800/myregistry/myapi:1",
			},
		},
		{
			name: "URL with Auth",
			args: args{
				destinationURL: urlWithAuth,
			},
			want: DestinationParams{
				AuthConfig: types.AuthConfig{
					Username: "username",
					Password: "password",
				},
				DestinationImageName: "registry.somebigbank.com:9800/myregistry/myapi:1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDestinationParams(tt.args.destinationURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDestinationParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildDestinationParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCLISaver_pushImage(t *testing.T) {
	goodURL, _ := url.Parse("docker://registry.fake/postgres:latest")
	type fields struct {
		Logger log.Logger
	}
	type args struct {
		ctx        context.Context
		progressCh chan interface{}
		saveOpts   SaveOpts
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				Logger: &logger2.TestLogger{T: t},
			},
			args: args{
				ctx:        context.Background(),
				progressCh: make(chan interface{}),
				saveOpts: SaveOpts{
					DestinationURL: goodURL,
					PullURL:        "postgres:latest",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mockimages.NewMockImageManager(gomock.NewController(t))
			s := &CLISaver{
				Logger: tt.fields.Logger,
				client: mockClient,
			}
			destinationParams, _ := buildDestinationParams(tt.args.saveOpts.DestinationURL)
			registryAuth, _ := makeAuthValue(destinationParams.AuthConfig)
			pushOpts := types.ImagePushOptions{
				RegistryAuth: registryAuth,
			}
			mockClient.EXPECT().ImageTag(tt.args.ctx, tt.args.saveOpts.PullURL, destinationParams.DestinationImageName).Return(nil)
			mockClient.EXPECT().ImagePush(tt.args.ctx, destinationParams.DestinationImageName, pushOpts).Return(ioutil.NopCloser(bytes.NewReader([]byte{})), nil)
			if err := s.pushImage(tt.args.ctx, tt.args.progressCh, tt.args.saveOpts); (err != nil) != tt.wantErr {
				t.Errorf("CLISaver.pushImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
