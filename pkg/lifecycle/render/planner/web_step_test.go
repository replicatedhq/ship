package planner

//
// import (
// 	"testing"
//
// 	"github.com/replicatedcom/ship/pkg/api"
// 	"github.com/replicatedhq/libyaml"
// )
//
// type TestPullWeb struct {
// 	Name         string
// 	Release      *api.Release
// 	ExpectedErr  bool
// 	ExpectedResp []byte
// }
//
// type TestWebAsset struct {
// 	Name        string
// 	Release     *api.Release
// 	ExpectedErr bool
// }
//
// func TestPullHelper(t *testing.T) {
// 	tests := []TestPullWeb{
// 		{
// 			Name: "empty",
// 			Release: &api.Release{
// 				Spec: api.Spec{
// 					Assets: api.Assets{
// 						V1: []api.Asset{{
// 							Web: &api.WebAsset{
// 								URL: "",
// 							},
// 						}},
// 					},
// 				},
// 			},
// 			ExpectedErr:  true,
// 			ExpectedResp: []byte(``),
// 		},
// 		// {
// 		// 	Name: "simple google",
// 		// 	Release: &api.Release{
// 		// 		Spec: api.Spec{
// 		// 			Assets: api.Assets{
// 		// 				V1: []api.Asset{{
// 		// 					Web: &api.WebAsset{
// 		// 						URL: "https://www.google.com",
// 		// 					},
// 		// 				}},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	ExpectedErr:  false,
// 		// 	ExpectedResp: []byte(``),
// 		// },
// 	}
//
// 	for _, test := range tests {
// 		t.Run(test.Name, func(t *testing.T) {
// 			// req := require.New(t)
// 			//
// 			// testAsset := test.Release.Spec.Assets.V1[0].Web
// 			//
// 			// body, err := pullWebAsset(testAsset)
// 			// if test.ExpectedErr {
// 			// 	req.Error(err)
// 			// } else {
// 			// 	req.Equal(body, test.ExpectedResp)
// 			// }
// 		})
// 	}
// }
//
// func TestWebAssetStep(t *testing.T) {
// 	tests := []TestWebAsset{
// 		{
// 			Name: "empty",
// 			Release: &api.Release{
// 				Spec: api.Spec{
// 					Assets: api.Assets{
// 						V1: []api.Asset{{
// 							Web: &api.WebAsset{
// 								URL: "",
// 								AssetShared: api.AssetShared{
// 									Dest:        "",
// 									Description: "",
// 								},
// 							},
// 						}},
// 					},
// 				},
// 			},
// 			ExpectedErr: true,
// 		},
// 		{
// 			Name: "empty",
// 			Release: &api.Release{
// 				Spec: api.Spec{
// 					Assets: api.Assets{
// 						V1: []api.Asset{{
// 							Web: &api.WebAsset{
// 								URL: "https://www.google.com",
// 								AssetShared: api.AssetShared{
// 									Dest:        "google.txt",
// 									Description: "",
// 								},
// 								Headers: map[string][]string{
// 									"Authorization": {`{{repl ConfigOption "authKey" }}`},
// 								},
// 							},
// 						}},
// 					},
// 					Config: api.Config{
// 						V1: []libyaml.ConfigGroup{{
// 							Name: "testing",
// 							Items: []*libyaml.ConfigItem{
// 								{
// 									Name:     "authKey",
// 									Value:    "abc123",
// 									Default:  "",
// 									Required: true,
// 									Hidden:   true,
// 								},
// 							},
// 						}},
// 					},
// 				},
// 			},
// 			ExpectedErr: false,
// 		},
// 		// {
// 		// 	Name: "simple google",
// 		// 	Release: &api.Release{
// 		// 		Spec: api.Spec{
// 		// 			Assets: api.Assets{
// 		// 				V1: []api.Asset{{
// 		// 					Web: &api.WebAsset{
// 		// 						URL: "https://www.google.com",
// 		// 						AssetShared: api.AssetShared{
// 		// 							Dest:        "./google.txt",
// 		// 							Description: "google",
// 		// 						},
// 		// 					},
// 		// 				}},
// 		// 			},
// 		// 		},
// 		// 	},
// 		// 	ExpectedErr: false,
// 		// },
// 	}
//
// 	for _, test := range tests {
// 		t.Run(test.Name, func(t *testing.T) {
// 			// req := require.New(t)
// 			//
// 			// mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
// 			// planner := &CLIPlanner{
// 			// 	Logger: log.NewNopLogger(),
// 			// 	UI:     cli.NewMockUi(),
// 			// 	Fs:     mockFS,
// 			// }
// 			//
// 			// asset := test.Release.Spec.Assets.V1[0].Web
// 			// config := test.Release.Spec.Config.V1
// 			//
// 			// step := planner.webStep(asset, config)
// 			//
// 			// executeErr := step.Execute(context.Background())
// 			//
// 			// if test.ExpectedErr {
// 			// 	req.Error(executeErr)
// 			// } else {
// 			// 	req.NoError(executeErr)
// 			//
// 			// 	// TODO: compare
// 			// 	_, readErr := mockFS.ReadFile(step.Dest)
// 			// 	req.NoError(readErr)
// 			// }
// 		})
// 	}
// }
