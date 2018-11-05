package terraform

import (
	"path"
	"testing"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

type filesForKubeConfig struct {
	clientCertificate    string
	clientKey            string
	clusterCACertificate string
	endpoint             string
}

func (f filesForKubeConfig) writeToFS(fs afero.Afero, dir string) error {
	if err := fs.WriteFile(path.Join(dir, "client_certificate"), []byte(f.clientCertificate), 0644); err != nil {
		return err
	}
	if err := fs.WriteFile(path.Join(dir, "client_key"), []byte(f.clientKey), 0644); err != nil {
		return err
	}
	if err := fs.WriteFile(path.Join(dir, "cluster_ca_certificate"), []byte(f.clusterCACertificate), 0644); err != nil {
		return err
	}
	if err := fs.WriteFile(path.Join(dir, "endpoint"), []byte(f.endpoint), 0644); err != nil {
		return err
	}
	return nil
}

func TestCreateKubeConfig(t *testing.T) {
	tests := []struct {
		releaseMetadata    api.ReleaseMetadata
		dir                string
		filesForKubeConfig filesForKubeConfig
		kubeConfigPath     string
		testAsset          *api.GKEAsset
		expectKubeConfig   kubeConfig
	}{
		{
			releaseMetadata: api.ReleaseMetadata{},
			dir:             "some-dir",
			filesForKubeConfig: filesForKubeConfig{
				clientCertificate:    "cert",
				clientKey:            "key",
				clusterCACertificate: "cluster-cert",
				endpoint:             "12.345.678.901",
			},
			kubeConfigPath: "my-shiny-new-kubeconfig",
			testAsset: &api.GKEAsset{
				ClusterName: "strawberry",
			},
			expectKubeConfig: kubeConfig{
				APIVersion: "v1",
				Kind:       "Config",
				Clusters: []namedKubeCluster{
					{
						Name: "ship-strawberry",
						Cluster: kubeCluster{
							CertificateAuthorityData: "cluster-cert",
							Server:                   "https://12.345.678.901",
						},
					},
				},
				Contexts: []namedKubeContext{
					{
						Name: "ship-strawberry",
						Context: kubeContext{
							Cluster:  "ship-strawberry",
							AuthInfo: "ship-strawberry",
						},
					},
				},
				AuthInfos: []namedKubeAuthInfo{
					{
						Name: "ship-strawberry",
						AuthInfo: kubeAuthInfo{
							ClientCertificateData: "cert",
							ClientKeyData:         "key",
						},
					},
				},
				CurrentContext: "ship-strawberry",
			},
		},
	}

	req := require.New(t)
	for _, tt := range tests {
		testLogger := &logger.TestLogger{T: t}
		v := viper.New()
		bb := templates.NewBuilderBuilder(testLogger, v)
		builder, err := bb.BaseBuilder(tt.releaseMetadata)
		req.NoError(err)
		templates.AddGoogleGKEPath(tt.testAsset.ClusterName, tt.kubeConfigPath)

		mockFS := afero.Afero{Fs: afero.NewMemMapFs()}
		terraformer := DaemonlessTerraformer{
			Logger: testLogger,
			FS:     mockFS,
		}
		err = tt.filesForKubeConfig.writeToFS(mockFS, tt.dir)
		req.NoError(err)

		err = terraformer.createKubeConfig(tt.dir, builder, tt.testAsset)
		req.NoError(err)

		kubeConfigB, err := terraformer.FS.ReadFile(path.Join(tt.dir, tt.kubeConfigPath))
		req.NoError(err)

		expectKubeConfigB, err := yaml.Marshal(tt.expectKubeConfig)
		req.Equal(string(expectKubeConfigB), string(kubeConfigB))
	}
}
