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

func Test_GetLicense(t *testing.T) {
	licenseID := "get-license-installation"

	var test = func() (err error) {
		req := require.New(t)

		v := viper.New()
		v.Set("customer-endpoint", fmt.Sprintf("http://localhost:%d/graphql", pact.Server.Port))

		gqlClient, err := replapp.NewGraphqlClient(v, http.DefaultClient)
		req.NoError(err)

		selector := replapp.Selector{
			LicenseID:     licenseID,
		}

		_, err = gqlClient.GetLicense(&selector)
		req.NoError(err)

		return nil
	}

	pact.AddInteraction().
		Given("A request to get a license").
		UponReceiving("A request to get the license from from licenseID").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/graphql"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", licenseID, ""))))),
				"Content-Type":  dsl.String("application/json"),
			},
			Body: map[string]interface{}{
				"operationName": "",
				"query":         replapp.GetLicenseQuery,
				"variables": map[string]interface{}{
					"licenseId": licenseID,
				},
			},
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"license": map[string]interface{}{
						"id": "get-license-installation",
						"assignee": "Get License - Customer 0",
						"createdAt": dsl.Like(dsl.String("Tue Jan 01 2019 01:23:46 GMT+0000 (Coordinated Universal Time)")),
						"expiresAt": nil,
						"type": dsl.Like(dsl.String("")),
					},
				},
			},
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
