package util

import (
	"bytes"

	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/pkg/resource"
)

func NewKubernetesResource(in []byte) (*resource.Resource, error) {
	var out unstructured.Unstructured

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(in), 1024)
	err := decoder.Decode(&out)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}

	return resource.NewResourceFromUnstruct(out), nil
}
