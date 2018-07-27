package images

import (
	"context"
	"reflect"
	"testing"

	"github.com/replicatedhq/ship/pkg/logger"
	"github.com/spf13/viper"

	"github.com/docker/docker/api/types"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func Test_buildDestinationParams(t *testing.T) {
	type args struct {
		destinationURL string
	}
	tests := []struct {
		name    string
		args    args
		want    DestinationParams
		wantErr bool
	}{
		{
			name: "Good URL",
			args: args{
				destinationURL: "docker://username:password@registry.somebigbank.com:9800/myregistry/myapi:1",
			},
			want: DestinationParams{
				AuthConfig: types.AuthConfig{
					Username: "username",
					Password: "password",
				},
				DestinationImageName: "registry.somebigbank.com:9800/myregistry/myapi:1",
			},
		},
		{
			name: "Bad URL",
			args: args{
				destinationURL: "docker://fdk432874*$&(#@&%*)sjlfkdsjflksdjf",
			},
			wantErr: true,
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
	type fields struct {
		Logger log.Logger
		client ImageManager
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
			name: "Fail",
			fields: fields{
				Logger: log.With(level.Debug(logger.FromViper(viper.GetViper()))),
				client: MockImageManager{},
			},
			args: args{
				ctx:        context.Background(),
				progressCh: make(chan interface{}),
				saveOpts: SaveOpts{
					DestinationURL: "*docker://registry.fake/postgres:latest",
					PullURL:        "postgres:latest",
				},
			},
			wantErr: true,
		},
		{
			name: "Success",
			fields: fields{
				Logger: log.With(level.Debug(logger.FromViper(viper.GetViper()))),
				client: MockImageManager{},
			},
			args: args{
				ctx:        context.Background(),
				progressCh: make(chan interface{}),
				saveOpts: SaveOpts{
					DestinationURL: "docker://registry.fake/postgres:latest",
					PullURL:        "postgres:latest",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CLISaver{
				Logger: tt.fields.Logger,
				client: tt.fields.client,
			}
			if err := s.pushImage(tt.args.ctx, tt.args.progressCh, tt.args.saveOpts); (err != nil) != tt.wantErr {
				t.Errorf("CLISaver.pushImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
