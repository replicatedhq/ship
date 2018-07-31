package state

type State interface {
	CurrentConfig() map[string]interface{}
	CurrentKustomize() *Kustomize
	CurrentHelmValues() string
}

var _ State = VersionedState{}
var _ State = Empty{}
var _ State = V0{}

type Empty struct{}

func (Empty) CurrentKustomize() *Kustomize          { return nil }
func (Empty) CurrentConfig() map[string]interface{} { return make(map[string]interface{}) }
func (Empty) CurrentHelmValues() string             { return "" }

type V0 map[string]interface{}

func (v V0) CurrentConfig() map[string]interface{} { return v }
func (v V0) CurrentKustomize() *Kustomize          { return nil }
func (v V0) CurrentHelmValues() string             { return "" }

type VersionedState struct {
	V1 *V1 `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

type V1 struct {
	Config     map[string]interface{} `json:"config" yaml:"config" hcl:"config"`
	Terraform  interface{}            `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	HelmValues string                 `json:"helmValues,omitempty" yaml:"helmValues,omitempty" hcl:"helmValues,omitempty"`
	Kustomize  *Kustomize             `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
}

type Overlay struct {
	Files             map[string]string `json:"files,omitempty" yaml:"files,omitempty" hcl:"files,omitempty"`
	KustomizationYAML string            `json:"kustomization_yaml,omitempty" yaml:"kustomization_yaml,omitempty" hcl:"kustomization_yaml,omitempty"`
}

type Kustomize struct {
	Overlays map[string]Overlay `json:"overlays,omitempty" yaml:"overlays,omitempty" hcl:"overlays,omitempty"`
}

func (u VersionedState) CurrentKustomize() *Kustomize {
	return u.V1.Kustomize
}

func (u VersionedState) CurrentConfig() map[string]interface{} {
	if u.V1 != nil && u.V1.Config != nil {
		return u.V1.Config
	}
	return make(map[string]interface{})
}

func (u VersionedState) CurrentHelmValues() string {
	return u.V1.HelmValues
}
