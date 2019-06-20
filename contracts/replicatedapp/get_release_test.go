package replicatedapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	replapp "github.com/replicatedhq/ship/pkg/specs/replicatedapp"
)

func Test_GetRelease(t *testing.T) {
	customerID := "ship-fetch-release-customer-0"
	installationID := "ship-fetch-release-installation-0"
	semver := "1.0.2"
	spec := `assets:
  v1:
    - inline:
        contents: |
	  #!/bin/bash
	  echo "installing nothing"
	  echo "config option: {{repl ConfigOption "test_option" }}"
	  dest: ./scripts/install.sh
	  mode: 0777
    - inline:
	contents: |
	  #!/bin/bash
	  echo "tested nothing"
	  echo "customer {{repl Installation "customer_id" }}"
	  echo "install {{repl Installation "installation_id" }}"
	  dest: ./scripts/test.sh
	  mode: 0777
config:
  v1:
    - name: test_options
      title: Test Options
      description: testing testing 123
      items:
	- name: test_option
	  title: Test Option
	  default: abc123_test-option-value
	  type: text

lifecycle:
  v1:
    - render: {}`
	var test = func() (err error) {
		req := require.New(t)

		v := viper.New()
		v.Set("customer-endpoint", fmt.Sprintf("http://localhost:%d/graphql", pact.Server.Port))

		gqlClient, err := replapp.NewGraphqlClient(v, http.DefaultClient)
		req.NoError(err)

		selector := replapp.Selector{
			CustomerID:     customerID,
			InstallationID: installationID,
			ReleaseSemver:  semver,
		}

		_, err = gqlClient.GetRelease(&selector)
		req.NoError(err)

		return nil
	}

	pact.AddInteraction().
		Given("A request to get a single app release").
		UponReceiving("A request to get the amn app release from a semver and customer id").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/graphql"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerID, installationID))))),
				"Content-Type":  dsl.String("application/json"),
			},
			Body: map[string]interface{}{
				"operationName": "",
				"query":         replapp.GetAppspecQuery,
				"variables": map[string]interface{}{
					"semver": semver,
				},
			},
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"shipRelease": map[string]interface{}{
						"id":   dsl.Like(dsl.String("generated")),
						"spec": dsl.Like(dsl.String(spec)),
					},
				},
			},
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
