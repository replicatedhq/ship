package e2e

import (
	"testing"

	"encoding/json"
	"io/ioutil"
	"net/url"

	"time"

	"context"

	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/ship"
	"github.com/stretchr/testify/require"
)

type CaseRunner struct {
	t        *testing.T
	req      *require.Assertions
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
	r.req.NoError(err)
	client := &GraphQLClient{
		GQLServer: gqlServer,
		Token:     r.testcase.config.Token,
	}

	spec, err := json.Marshal(r.testcase.Spec)
	r.req.NoError(err)

	_, err = client.PromoteRelease(
		string(spec),
		r.testcase.config.ChannelID,
		r.testcase.config.Semver,
		`Integration test run on `+time.Now().String(),
	)
	r.req.NoError(err)

}

func (r *CaseRunner) Run() {
	r.promoteRelease()
	r.runShipForCustomer()
	r.validateFiles()
}

func (r *CaseRunner) runShipForCustomer() {
	s, err := ship.Get()
	r.req.NoError(err)

	err = s.Execute(context.Background())
	r.req.NoError(err)
}
func (r *CaseRunner) validateFiles() {
	for path, expected := range r.testcase.Expect {
		actual, err := ioutil.ReadFile(path)
		r.req.NoError(err)
		r.req.Equal(expected, string(actual))
	}

}
