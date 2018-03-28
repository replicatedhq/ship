package e2e

import (
	"testing"

	"context"
	"encoding/json"
	"io/ioutil"
	"net/url"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/ship"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type CaseRunner struct {
	t        *testing.T
	assert   *require.Assertions
	testcase testcase
}

type testcase struct {
	Name   string
	Spec   *api.Spec
	Expect map[string]string
	config Config
}

func (r *CaseRunner) promoteRelease() {

	gqlServer, err := url.Parse(r.testcase.config.GQL)
	r.assert.NoError(err)
	client := &GraphQLClient{
		GQLServer: gqlServer,
		Token:     r.testcase.config.Token,
		assert:    r.assert,
	}

	spec, err := json.Marshal(r.testcase.Spec)
	r.assert.NoError(err)

	client.promoteRelease(
		string(spec),
		r.testcase.config.ChannelID,
		r.testcase.config.Semver,
	)

}

func (r *CaseRunner) Run() {

	r.promoteRelease()
	r.runShipForCustomer()
	r.validateFiles()

}
func (r *CaseRunner) runShipForCustomer() {
	// todo do each testcase in its own tmp directory,
	// also maybe fork or docker run or something
	s, err := ship.FromViper(viper.GetViper())
	r.assert.NoError(err)

	err = s.Execute(context.Background())
	r.assert.NoError(err)
}
func (r *CaseRunner) validateFiles() {
	for path, expected := range r.testcase.Expect {
		actual, err := ioutil.ReadFile(path)
		r.assert.NoError(err)
		r.assert.Equal(expected, string(actual))
	}

}
