package e2e

import (
	"testing"

	"io/ioutil"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type Runner struct {
	v      *viper.Viper
	t      *testing.T
	assert *require.Assertions
}

type Config struct {
	GQL        string
	Token      string
	Semver     string
	CustomerID string
	ChannelID  string
}

func (r *Runner) Run(t *testing.T) {
	r.t = t
	r.assert = require.New(t)

	config := Config{
		GQL:        r.requireOption("graphql-api-address"),
		Token:      r.requireOption("vendor-token"),
		Semver:     r.requireOption("release-semver"),
		CustomerID: r.requireOption("customer-id"),
		ChannelID:  r.requireOption("channel-id"),
	}

	cases := r.loadTestCases("deploy/e2e.yml")
	for _, test := range cases {
		test.config = config
		r.t.Run(test.Name, r.TestCase(test))
	}
}

func (r *Runner) TestCase(test testcase) func(t *testing.T) {
	return func(t *testing.T) {
		(&CaseRunner{
			t:        t,
			assert:   require.New(t),
			testcase: test,
		}).Run()
	}
}

func (r *Runner) requireOption(name string) string {
	opt := viper.GetString(name)
	r.assert.NotEmpty(opt, name)
	return opt
}

func (r *Runner) loadTestCases(path string) []testcase {
	tests := make([]testcase, 1)
	contents, err := ioutil.ReadFile(path)
	r.assert.NoError(err)
	err = yaml.UnmarshalStrict(contents, &tests)
	r.assert.NoError(err)
	return tests
}
