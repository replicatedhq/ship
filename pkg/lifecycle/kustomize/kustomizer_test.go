package kustomize

import (
	"context"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/types"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/lifecycle/daemon/daemontypes"
	"github.com/replicatedhq/ship/pkg/state"
	daemon2 "github.com/replicatedhq/ship/pkg/test-mocks/daemon"
	state2 "github.com/replicatedhq/ship/pkg/test-mocks/state"
	"github.com/replicatedhq/ship/pkg/testing/logger"
)

const minimalValidYaml = `
kind: Deployment
metadata:
  name: myDeployment
`

func Test_kustomizer_writePatches(t *testing.T) {
	destDir := path.Join("overlays", "ship")

	type args struct {
		shipOverlay state.Overlay
		destDir     string
	}
	tests := []struct {
		name        string
		args        args
		expectFiles map[string]string
		want        []types.PatchStrategicMerge
		wantErr     bool
	}{
		{
			name: "No patches in state",
			args: args{
				shipOverlay: state.Overlay{
					Patches: map[string]string{},
				},
				destDir: destDir,
			},
			expectFiles: map[string]string{},
			want:        nil,
		},
		{
			name: "Patches in state",
			args: args{
				shipOverlay: state.Overlay{
					Patches: map[string]string{
						"a.yaml":         "---",
						"/folder/b.yaml": "---",
					},
				},
				destDir: destDir,
			},
			expectFiles: map[string]string{
				"a.yaml":        "---",
				"folder/b.yaml": "---",
			},
			want: []types.PatchStrategicMerge{"a.yaml", "folder/b.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			mockDaemon := daemon2.NewMockDaemon(mc)
			mockState := state2.NewMockManager(mc)

			// need a real FS because afero.Rename on a memMapFs doesn't copy directories recursively
			fs := afero.Afero{Fs: afero.NewOsFs()}
			tmpdir, err := fs.TempDir("./", tt.name)
			req.NoError(err)
			defer fs.RemoveAll(tmpdir) // nolint: errcheck

			mockFs := afero.Afero{Fs: afero.NewBasePathFs(afero.NewOsFs(), tmpdir)}
			// its chrooted to a temp dir, but this needs to exist
			err = mockFs.MkdirAll(".ship/tmp/", 0755)
			req.NoError(err)
			l := &daemonkustomizer{
				Kustomizer: Kustomizer{
					Logger: testLogger,
					State:  mockState,
					FS:     mockFs,
				},
				Daemon: mockDaemon,
			}

			got, err := l.writePatches(mockFs, tt.args.shipOverlay, tt.args.destDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("kustomizer.writePatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, filename := range tt.want {
				req.Contains(got, filename)
			}

			for file, contents := range tt.expectFiles {
				fileBytes, err := l.FS.ReadFile(path.Join(destDir, file))
				if err != nil {
					t.Errorf("expected file at %v, received error instead: %v", file, err)
				}
				req.Equal(contents, string(fileBytes))
			}
		})
	}
}

func Test_kustomizer_writeOverlay(t *testing.T) {
	mockStep := api.Kustomize{
		Base:    constants.KustomizeBasePath,
		Overlay: path.Join("overlays", "ship"),
	}

	tests := []struct {
		name                  string
		relativePatchPaths    []types.PatchStrategicMerge
		existingKustomization types.Kustomization
		expectFile            string
		wantErr               bool
	}{
		{
			name:               "No patches",
			relativePatchPaths: []types.PatchStrategicMerge{},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../defaults
`,
		},
		{
			name:               "Patches provided",
			relativePatchPaths: []types.PatchStrategicMerge{"a.yaml", "b.yaml", "c.yaml"},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
patchesStrategicMerge:
- a.yaml
- b.yaml
- c.yaml
bases:
- ../defaults
`,
		},
		{
			name:               "No patches but existing kustomization",
			relativePatchPaths: []types.PatchStrategicMerge{},
			existingKustomization: types.Kustomization{
				TypeMeta: types.TypeMeta{Kind: types.KustomizationKind, APIVersion: types.KustomizationVersion},
				PatchesJson6902: []types.PatchJson6902{
					{
						Path: "abc.json",
						Target: &types.PatchTarget{
							Gvk: gvk.Gvk{
								Group:   "groupa",
								Version: "versionb",
								Kind:    "kindc",
							},
							Namespace: "nsd",
							Name:      "namee",
						},
					},
				},
			},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
patchesJson6902:
- target:
    group: groupa
    version: versionb
    kind: kindc
    namespace: nsd
    name: namee
  path: abc.json
bases:
- ../defaults
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			mockDaemon := daemon2.NewMockDaemon(mc)
			mockState := state2.NewMockManager(mc)
			mockFs := afero.Afero{Fs: afero.NewMemMapFs()}

			l := &daemonkustomizer{
				Kustomizer: Kustomizer{
					Logger: testLogger,
					State:  mockState,
					FS:     mockFs,
				},
				Daemon: mockDaemon,
			}
			if err := l.writeOverlay(mockStep, tt.relativePatchPaths, nil, tt.existingKustomization); (err != nil) != tt.wantErr {
				t.Errorf("kustomizer.writeOverlay() error = %v, wantErr %v", err, tt.wantErr)
			}

			overlayPathDest := path.Join(mockStep.OverlayPath(), "kustomization.yaml")
			fileBytes, err := l.FS.ReadFile(overlayPathDest)
			if err != nil {
				t.Errorf("expected file at %v, received error instead: %v", overlayPathDest, err)
			}
			req.Equal(tt.expectFile, string(fileBytes))
		})
	}
}

func Test_kustomizer_writeBase(t *testing.T) {
	mockStep := api.Kustomize{
		Base:    constants.KustomizeBasePath,
		Overlay: path.Join("overlays", "ship"),
	}

	type fields struct {
		GetFS func() (afero.Afero, error)
	}
	tests := []struct {
		name          string
		fields        fields
		expectFile    string
		wantErr       bool
		excludedBases []string
	}{
		{
			name: "No base files",
			fields: fields{
				GetFS: func() (afero.Afero, error) {
					fs := afero.Afero{Fs: afero.NewMemMapFs()}
					err := fs.Mkdir(constants.KustomizeBasePath, 0777)
					if err != nil {
						return afero.Afero{}, err
					}
					return fs, nil
				},
			},
			wantErr: true,
		},
		{
			name: "Flat base files",
			fields: fields{
				GetFS: func() (afero.Afero, error) {
					fs := afero.Afero{Fs: afero.NewMemMapFs()}
					if err := fs.Mkdir(constants.KustomizeBasePath, 0777); err != nil {
						return afero.Afero{}, err
					}

					files := []string{"a.yaml", "b.yaml", "c.yaml"}
					for _, file := range files {
						if err := fs.WriteFile(
							path.Join(constants.KustomizeBasePath, file),
							[]byte(minimalValidYaml),
							0777,
						); err != nil {
							return afero.Afero{}, err
						}
					}

					return fs, nil
				},
			},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- a.yaml
- b.yaml
- c.yaml
`,
		},
		{
			name: "Base files with nested chart",
			fields: fields{
				GetFS: func() (afero.Afero, error) {
					fs := afero.Afero{Fs: afero.NewMemMapFs()}
					nestedChartPath := path.Join(
						constants.KustomizeBasePath,
						"charts/kube-stats-metrics/templates",
					)
					if err := fs.MkdirAll(nestedChartPath, 0777); err != nil {
						return afero.Afero{}, err
					}

					files := []string{
						"deployment.yaml",
						"clusterrole.yaml",
						"charts/kube-stats-metrics/templates/deployment.yaml",
					}
					for _, file := range files {
						if err := fs.WriteFile(
							path.Join(constants.KustomizeBasePath, file),
							[]byte(minimalValidYaml),
							0777,
						); err != nil {
							return afero.Afero{}, err
						}
					}

					return fs, nil
				},
			},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- charts/kube-stats-metrics/templates/deployment.yaml
- clusterrole.yaml
- deployment.yaml
`,
		},
		{
			name: "Base files with nested and excluded chart",
			fields: fields{
				GetFS: func() (afero.Afero, error) {
					fs := afero.Afero{Fs: afero.NewMemMapFs()}
					nestedChartPath := path.Join(
						constants.KustomizeBasePath,
						"charts/kube-stats-metrics/templates",
					)
					if err := fs.MkdirAll(nestedChartPath, 0777); err != nil {
						return afero.Afero{}, err
					}

					files := []string{
						"deployment.yaml",
						"clusterrole.yaml",
						"charts/kube-stats-metrics/templates/deployment.yaml",
					}
					for _, file := range files {
						if err := fs.WriteFile(
							path.Join(constants.KustomizeBasePath, file),
							[]byte(minimalValidYaml),
							0777,
						); err != nil {
							return afero.Afero{}, err
						}
					}

					return fs, nil
				},
			},
			expectFile: `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- charts/kube-stats-metrics/templates/deployment.yaml
- deployment.yaml
`,
			excludedBases: []string{"/clusterrole.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			mockDaemon := daemon2.NewMockDaemon(mc)
			mockState := state2.NewMockManager(mc)

			mockState.EXPECT().CachedState().Return(state.State{
				V1: &state.V1{
					Kustomize: &state.Kustomize{
						Overlays: map[string]state.Overlay{
							"ship": state.Overlay{
								ExcludedBases: tt.excludedBases,
							},
						},
					},
				},
			}, nil).AnyTimes()

			fs, err := tt.fields.GetFS()
			req.NoError(err)

			l := &daemonkustomizer{
				Kustomizer: Kustomizer{
					Logger: testLogger,
					State:  mockState,
					FS:     fs,
				},
				Daemon: mockDaemon,
			}

			if err := l.writeBase(mockStep.Base); (err != nil) != tt.wantErr {
				t.Errorf("kustomizer.writeBase() error = %v, wantErr %v", err, tt.wantErr)
			} else if err == nil {
				basePathDest := path.Join(mockStep.Base, "kustomization.yaml")
				fileBytes, err := l.FS.ReadFile(basePathDest)
				if err != nil {
					t.Errorf("expected file at %v, received error instead: %v", basePathDest, err)
				}
				req.Equal(tt.expectFile, string(fileBytes))
			}
		})
	}
}

func TestKustomizer(t *testing.T) {
	tests := []struct {
		name        string
		kustomize   *state.Kustomize
		expectFiles map[string]string
	}{
		{
			name:      "no files",
			kustomize: nil,
			expectFiles: map[string]string{
				"overlays/ship/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../defaults
`,
				"base/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- deployment.yaml
`,
			},
		},
		{
			name: "one file",
			kustomize: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Patches: map[string]string{
							"/deployment.yaml": `---
metadata:
  name: my-deploy
spec:
  replicas: 100`,
						},
					},
				},
			},
			expectFiles: map[string]string{
				"overlays/ship/deployment.yaml": `---
metadata:
  name: my-deploy
spec:
  replicas: 100`,

				"overlays/ship/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
patchesStrategicMerge:
- deployment.yaml
bases:
- ../defaults
`,
				"base/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- deployment.yaml
`,
			},
		},
		{
			name: "adding a resource",
			kustomize: &state.Kustomize{
				Overlays: map[string]state.Overlay{
					"ship": {
						Resources: map[string]string{
							"/limitrange.yaml": `---
apiVersion: v1
kind: LimitRange
metadata:
  name: mem-limit-range
spec:
  limits:
  - default:
      memory: 512Mi
    defaultRequest:
      memory: 256Mi
    type: Container`,
						},
					},
				},
			},
			expectFiles: map[string]string{
				"overlays/ship/limitrange.yaml": `---
apiVersion: v1
kind: LimitRange
metadata:
  name: mem-limit-range
spec:
  limits:
  - default:
      memory: 512Mi
    defaultRequest:
      memory: 256Mi
    type: Container`,

				"overlays/ship/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- limitrange.yaml
bases:
- ../defaults
`,
				"base/kustomization.yaml": `kind: Kustomization
apiVersion: kustomize.config.k8s.io/v1beta1
resources:
- deployment.yaml
`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			mc := gomock.NewController(t)
			testLogger := &logger.TestLogger{T: t}
			mockDaemon := daemon2.NewMockDaemon(mc)
			mockState := state2.NewMockManager(mc)

			mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
			err := mockFS.Mkdir(constants.KustomizeBasePath, 0777)
			req.NoError(err)

			err = mockFS.WriteFile(
				path.Join(constants.KustomizeBasePath, "deployment.yaml"),
				[]byte(minimalValidYaml),
				0666,
			)
			req.NoError(err)

			saveChan := make(chan interface{})
			close(saveChan)

			ctx := context.Background()
			release := api.Release{}

			mockDaemon.EXPECT().EnsureStarted(ctx, &release)
			mockDaemon.EXPECT().PushKustomizeStep(ctx, daemontypes.Kustomize{
				BasePath: constants.KustomizeBasePath,
			})
			mockDaemon.EXPECT().KustomizeSavedChan().Return(saveChan)
			mockState.EXPECT().CachedState().Return(state.State{V1: &state.V1{
				Kustomize: test.kustomize,
			}}, nil).Times(2)

			k := &daemonkustomizer{
				Kustomizer: Kustomizer{
					Logger: testLogger,
					FS:     mockFS,
					State:  mockState,
				},
				Daemon: mockDaemon,
			}

			err = k.Execute(
				ctx,
				&release,
				api.Kustomize{
					Base:    constants.KustomizeBasePath,
					Overlay: "overlays/ship",
				},
			)

			for name, contents := range test.expectFiles {
				actual, err := mockFS.ReadFile(name)
				req.NoError(err, "read expected file %s", name)
				req.Equal(contents, string(actual))
			}

			req.NoError(err)
		})
	}
}
