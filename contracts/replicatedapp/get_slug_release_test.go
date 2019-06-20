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

func Test_GetSlugRelease(t *testing.T) {
	appSlug := "get-slug-release-app"
	licenseID := "get-slug-release-installation"
	releaseID := "get-slug-release-app-release"
	semver := "get-slug-release-semver"

	var test = func() (err error) {
		req := require.New(t)

		v := viper.New()
		v.Set("customer-endpoint", fmt.Sprintf("http://localhost:%d/graphql", pact.Server.Port))

		gqlClient, err := replapp.NewGraphqlClient(v, http.DefaultClient)
		req.NoError(err)

		selector := replapp.Selector{
			AppSlug:       appSlug,
			LicenseID:     licenseID,
			ReleaseID:     releaseID,
			ReleaseSemver: semver,
		}

		_, err = gqlClient.GetSlugRelease(&selector)
		req.NoError(err)

		return nil
	}

	pact.AddInteraction().
		Given("A request to get a slug release").
		UponReceiving("A request to get the slug release from appSlug, licenseID, releaseID and semver").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/graphql"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", licenseID, ""))))),
				"Content-Type":  dsl.String("application/json"),
			},
			Body: map[string]interface{}{
				"operationName": "",
				"query":         replapp.GetSlugAppSpecQuery,
				"variables": map[string]interface{}{
					"appSlug":   appSlug,
					"licenseID": licenseID,
					"releaseID": releaseID,
					"semver":    semver,
				},
			},
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"shipSlugRelease": map[string]interface{}{
						"id": "get-slug-release-app-release",
						"sequence": 1,
						"channelId": "get-slug-release-app-stable",
						"channelName": "Stable",
						"channelIcon": nil,
						"semver": "1.0.2",
						"releaseNotes": "1.0.2",
						"spec": dsl.Like(dsl.String("assets:\n  v1:\n    - inline:\n        contents: |\n          #!/bin/bash\n          echo \"installing nothing\"\n          echo \"config option: {{repl ConfigOption \"test_option\" }}\"\n        dest: ./scripts/install.sh\n        mode: 0777\n    - inline:\n        contents: |\n          #!/bin/bash\n          echo \"tested nothing\"\n          echo \"customer {{repl Installation \"customer_id\" }}\"\n          echo \"install {{repl Installation \"installation_id\" }}\"\n        dest: ./scripts/test.sh\n        mode: 0777\nconfig:\n  v1:\n    - name: test_options\n      title: Test Options\n      description: testing testing 123\n      items:\n      - name: test_option\n        title: Test Option\n        default: abc123_test-option-value\n        type: text\nlifecycle:\n  v1:\n    - render: {}\n")),
						"images": []string{},
						"githubContents": []string{},
						"entitlements": map[string]interface{}{
							"values": []string{},
							"utilizations": []string{},
							"meta": map[string]interface{}{
								"lastUpdated": dsl.Like(dsl.String("Wed Jun 19 2019 21:12:23 GMT+0000 (Coordinated Universal Time)")),
								"customerID": "get-slug-release-customer-0",
								"installationID": nil,
							},
							"serialized": dsl.Like(dsl.String("{\"customerID\":\"get-slug-release-customer-0\",\"lastUpdated\":\"2019-06-19T21:12:23.173Z\"}[]")),
							"signature": dsl.Like(dsl.String("L5kLCgGP2OBZ0X54kZnOi1/4hG0geh/8JnQwV56fUwov9en9vrUscdK5LlIEw7PD8sWscqHZ7iNWgfJYKdTW2+zM+JgKAfQke02+Abx7qstjqJqZf/3RqeQweIIbppNLKRga4waD53br6REAyND6CQZENBz5ZXYSO6aJS+neblmICHn//4sbHTVv0SVz0szXcElvWztiZeLEekpl8xShCmCc23rlWgzml53UDPSM6E15/Erbryx4tzabJjxgFmGub3qp7b2c3eMPMVyFQp5fVvLGqsOXiBgvnpEWcn12e+vHks1gFhDyx9NtuVMjpYgeYXP3KHMOoAVrulREa2vAxGEOxYdRRjBATd5wm//fr9N4rY/WTH2oAesmHzizDGirD4wJYkHPKz3qv6hzB6D2Qix3+y/4dhO9nW8D6skK/c1XLTq4uhswrxT71xOuRU/c22Ee78Gy+Nq6RGigjclSFCWRABYP0MIoWKWH3UgW6RuSOyojEpbX2Wif9orZ3tenu1ne40zzZPQ5mDC6uOlDbUBg9GOB/QqEA90K5M9avBM14RWBXdJdFdmpXFiPtRdKzu2gsEjaMQVutI9dXe8M4U1Z1w5TIuym/1cXiNFe+YQBrUdk7bXUl7g+fj7kjtFqONm9DhR718IsdR0+QhodqglWwOTjDYXuIdGnrknG8n0=")),
						},
						"created": dsl.Like(dsl.String("Tue Jan 01 2019 02:23:46 GMT+0000 (Coordinated Universal Time)")),
						"registrySecret": dsl.Like(dsl.String("3bfd99a69b5748fab756a593c7dcc852")),
					},
				},
			},
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
