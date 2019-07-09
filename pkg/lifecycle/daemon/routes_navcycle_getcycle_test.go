package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"testing"

	"github.com/go-test/deep"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

type lifecycleTestcase struct {
	Name         string         `yaml:"name"`
	Lifecycle    []api.Step     `yaml:"lifecycle"`
	ExpectStatus int            `yaml:"expectStatus"`
	ExpectBody   []lifeycleStep `yaml:"expectBody"`
}

func TestNavcycle(t *testing.T) {
	tests := loadTestCases(t)
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)
			release := &api.Release{
				Spec: api.Spec{
					Lifecycle: api.Lifecycle{
						V1: test.Lifecycle,
					},
				},
			}
			testLogger := &logger.TestLogger{T: t}
			v2 := &NavcycleRoutes{
				Logger:       testLogger,
				StepProgress: &daemontypes.ProgressMap{},
			}

			func() {
				_, port, cancelFunc, err := initTestDaemon(t, release, v2)
				defer cancelFunc()
				req.NoError(err)
				addr := fmt.Sprintf("http://localhost:%d", port)
				req := require.New(t)

				testGet(addr, test, req)

			}()
		})
	}
}

func testGet(
	addr string,
	test lifecycleTestcase,
	req *require.Assertions,
) {
	resp, err := http.Get(fmt.Sprintf("%s%s", addr, "/api/v1/navcycle"))
	req.NoError(err)
	req.Equal(resp.StatusCode, test.ExpectStatus)
	bytes, err := ioutil.ReadAll(resp.Body)
	req.NoError(err)
	var deserializeTarget []lifeycleStep
	err = json.Unmarshal(bytes, &deserializeTarget)
	req.NoError(err)

	diff := deep.Equal(test.ExpectBody, deserializeTarget)
	bodyForDebug, err := json.Marshal(test.ExpectBody)
	req.NoError(err)
	req.Empty(diff, "\nexpect: %s\nactual: %s", bodyForDebug, string(bytes))
}

func loadTestCases(t *testing.T) []lifecycleTestcase {
	contents, err := ioutil.ReadFile(path.Join("test-cases", "routes_v2_lifecycle.yml"))
	require.NoError(t, err, "load test cases")

	cases := make([]lifecycleTestcase, 1)
	err = yaml.Unmarshal(contents, &cases)

	require.NoError(t, err, "unmarshal test cases")

	return cases
}
