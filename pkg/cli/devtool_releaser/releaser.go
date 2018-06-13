package devtool_releaser

import (
	"context"

	"io/ioutil"
	"net/url"

	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/e2e"
	"github.com/spf13/viper"
)

type Releaser struct {
	viper  *viper.Viper
	logger log.Logger
	ui     cli.Ui
}

func (r *Releaser) Release(
	ctx context.Context,
) error {

	token := r.viper.GetString("vendor-token")
	if token == "" {
		return errors.New("param vendor-token is required")
	}

	specFile := r.viper.GetString("spec-file")
	if specFile == "" {
		return errors.New("param spec-file is required")
	}

	specContents, err := ioutil.ReadFile(specFile)
	if err != nil {
		return errors.Wrapf(err, "read spec-file \"%s\"", specFile)
	}

	semver := r.viper.GetString("semver")
	if semver == "" {
		return errors.New("param semver is required")
	}

	channelId := r.viper.GetString("channel-id")
	if channelId == "" {
		return errors.New("param channel-id is required")
	}

	gqlAddr := r.viper.GetString("graphql-api-address")
	gqlServer, err := url.Parse(gqlAddr)
	if err != nil {
		return errors.Wrapf(err, "parse graphql-api-address URL \"%s\"", gqlAddr)
	}
	client := &e2e.GraphQLClient{
		GQLServer: gqlServer,
		Token:     token,
		Logger:    r.logger,
	}

	data, err := client.PromoteRelease(
		string(specContents),
		channelId,
		semver,
		r.viper.GetString("release-notes"),
	)

	r.ui.Info(fmt.Sprintf("received data %+v", data))

	if err != nil {
		return errors.Wrapf(err, "promote release")
	}

	return nil
}
