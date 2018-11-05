package terraform

import (
	"fmt"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"gopkg.in/yaml.v2"
)

// A mostly minimal representation of a kube config as per -
// https://github.com/kubernetes/client-go/blob/19c591bac28a94ca793a2f18a0cf0f2e800fad04/tools/clientcmd/api/types.go
type kubeConfig struct {
	APIVersion     string              `json:"apiVersion" yaml:"apiVersion"`
	Kind           string              `json:"kind" yaml:"kind"`
	Clusters       []namedKubeCluster  `json:"clusters" yaml:"clusters"`
	Contexts       []namedKubeContext  `json:"contexts" yaml:"contexts"`
	AuthInfos      []namedKubeAuthInfo `json:"users" yaml:"users"`
	CurrentContext string              `json:"current-context" yaml:"current-context"`
}

type namedKubeCluster struct {
	Name    string      `json:"name" yaml:"name"`
	Cluster kubeCluster `json:"cluster" yaml:"cluster"`
}

type kubeCluster struct {
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	// +optional
	CertificateAuthorityData string `json:"certificate-authority-data" yaml:"certificate-authority-data"`
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server" yaml:"server"`
}

type namedKubeContext struct {
	Name    string      `json:"name" yaml:"name"`
	Context kubeContext `json:"context" yaml:"context"`
}

type kubeContext struct {
	// Cluster is the name of the cluster for this context
	Cluster string `json:"cluster" yaml:"cluster"`
	// AuthInfo is the name of the authInfo for this context
	AuthInfo string `json:"user" yaml:"user"`
}

type namedKubeAuthInfo struct {
	Name     string       `json:"name" yaml:"name"`
	AuthInfo kubeAuthInfo `json:"user" yaml:"user"`
}

type kubeAuthInfo struct {
	// ClientCertificateData contains PEM-encoded data from a client cert file for TLS. Overrides ClientCertificate
	// +optional
	ClientCertificateData string `json:"client-certificate-data" yaml:"client-certificate-data"`
	// ClientKeyData contains PEM-encoded data from a client key file for TLS. Overrides ClientKey
	// +optional
	ClientKeyData string `json:"client-key-data" yaml:"client-key-data"`
}

// createKubeConfig creates a kubeconfig to be used connect to a newly terraformed GKE cluster
// Requires the cluster CertificateAuthorityData, Server URL, ClientCertificateData, and ClientKeyData.
func (t *DaemonlessTerraformer) createKubeConfig(dir string, builder *templates.Builder, gkeAsset *api.GKEAsset) error {
	debug := level.Debug(log.With(t.Logger, "struct", "ForkTerraformer", "method", "createKubeConfig"))

	builtKubePath, err := builder.String(fmt.Sprintf(`{{repl GoogleGKE "%s"}}`, gkeAsset.ClusterName))
	if err != nil {
		return errors.Wrap(err, "build kubepath")
	}

	debug.Log("event", "read client_certificate", "dest", path.Join(dir, "client_certificate"))
	clientCert, err := t.FS.ReadFile(path.Join(dir, "client_certificate"))
	if err != nil {
		return errors.Wrap(err, "read client_certificate")
	}
	debug.Log("event", "read client_key", "dest", path.Join(dir, "client_key"))
	clientKey, err := t.FS.ReadFile(path.Join(dir, "client_key"))
	if err != nil {
		return errors.Wrap(err, "read client_key")
	}
	debug.Log("event", "read cluster_ca_certificate", "dest", path.Join(dir, "cluster_ca_certificate"))
	clusterCA, err := t.FS.ReadFile(path.Join(dir, "cluster_ca_certificate"))
	if err != nil {
		return errors.Wrap(err, "read cluster_ca_certificate")
	}
	debug.Log("event", "read endpoint", "dest", path.Join(dir, "endpoint"))
	endpoint, err := t.FS.ReadFile(path.Join(dir, "endpoint"))
	if err != nil {
		return errors.Wrap(err, "read endpoint")
	}

	debug.Log("event", "create new kube config")
	newConfig := kubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
		Clusters: []namedKubeCluster{
			{
				Name: fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
				Cluster: kubeCluster{
					CertificateAuthorityData: string(clusterCA),
					Server:                   fmt.Sprintf("https://%s", string(endpoint)),
				},
			},
		},
		Contexts: []namedKubeContext{
			{
				Name: fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
				Context: kubeContext{
					Cluster:  fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
					AuthInfo: fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
				},
			},
		},
		AuthInfos: []namedKubeAuthInfo{
			{
				Name: fmt.Sprintf("ship-%s", gkeAsset.ClusterName),
				AuthInfo: kubeAuthInfo{
					ClientCertificateData: string(clientCert),
					ClientKeyData:         string(clientKey),
				},
			},
		},
	}

	debug.Log("event", "marshal new kube config")
	newConfigB, err := yaml.Marshal(newConfig)
	if err != nil {
		return errors.Wrap(err, "marshal new kube config")
	}

	debug.Log("event", "mkdir for kube config", "dest", path.Join(dir, path.Dir(builtKubePath)))
	if err := t.FS.MkdirAll(path.Join(dir, path.Dir(builtKubePath)), 0777); err != nil {
		return errors.Wrap(err, "mkdir for kube config")
	}

	debug.Log("event", "write new kube config", "dest", path.Join(dir, builtKubePath))
	if err := t.FS.WriteFile(path.Join(dir, builtKubePath), newConfigB, 0666); err != nil {
		return errors.Wrap(err, "write new kube config")
	}

	return nil
}
