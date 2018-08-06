package state

type State interface {
	CurrentConfig() map[string]interface{}
	CurrentKustomize() *Kustomize
	CurrentKustomizeOverlay(filename string) string
	CurrentHelmValues() string
	CurrentChartURL() string
	Versioned() VersionedState
}

var _ State = VersionedState{}
var _ State = Empty{}
var _ State = V0{}

type Empty struct{}

func (Empty) CurrentKustomize() *Kustomize          { return nil }
func (Empty) CurrentKustomizeOverlay(string) string { return "" }
func (Empty) CurrentConfig() map[string]interface{} { return make(map[string]interface{}) }
func (Empty) CurrentHelmValues() string             { return "" }
func (Empty) CurrentChartURL() string               { return "" }
func (Empty) Versioned() VersionedState             { return VersionedState{V1: &V1{}} }

type V0 map[string]interface{}

func (v V0) CurrentConfig() map[string]interface{} { return v }
func (v V0) CurrentKustomize() *Kustomize          { return nil }
func (v V0) CurrentKustomizeOverlay(string) string { return "" }
func (v V0) CurrentHelmValues() string             { return "" }
func (v V0) CurrentChartURL() string               { return "" }
func (v V0) Versioned() VersionedState             { return VersionedState{V1: &V1{Config: v}} }

type VersionedState struct {
	V1 *V1 `json:"v1,omitempty" yaml:"v1,omitempty" hcl:"v1,omitempty"`
}

type V1 struct {
	Config     map[string]interface{} `json:"config" yaml:"config" hcl:"config"`
	Terraform  interface{}            `json:"terraform,omitempty" yaml:"terraform,omitempty" hcl:"terraform,omitempty"`
	HelmValues string                 `json:"helmValues,omitempty" yaml:"helmValues,omitempty" hcl:"helmValues,omitempty"`
	Kustomize  *Kustomize             `json:"kustomize,omitempty" yaml:"kustomize,omitempty" hcl:"kustomize,omitempty"`
	ChartURL   string                 `json:"chartURL,omitempty" yaml:"chartURL,omitempty" hcl:"chartURL,omitempty"`
}

type Overlay struct {
	Patches           map[string]string `json:"patches,omitempty" yaml:"patches,omitempty" hcl:"patches,omitempty"`
	KustomizationYAML string            `json:"kustomization_yaml,omitempty" yaml:"kustomization_yaml,omitempty" hcl:"kustomization_yaml,omitempty"`
}

type Kustomize struct {
	Overlays map[string]Overlay `json:"overlays,omitempty" yaml:"overlays,omitempty" hcl:"overlays,omitempty"`
}

func (k *Kustomize) Ship() Overlay {
	if k.Overlays == nil {
		return Overlay{}
	}
	if ship, ok := k.Overlays["ship"]; ok {
		return ship
	}

	return Overlay{}
}

func (u VersionedState) CurrentKustomize() *Kustomize {
	return u.V1.Kustomize
}

func (u VersionedState) CurrentKustomizeOverlay(filename string) string {
	if u.V1.Kustomize == nil {
		return ""
	}

	if u.V1.Kustomize.Overlays == nil {
		return ""
	}

	overlay, ok := u.V1.Kustomize.Overlays["ship"]
	if !ok {
		return ""
	}

	if overlay.Patches == nil {
		return ""
	}

	file, ok := overlay.Patches[filename]
	if ok {
		return file
	}

	return ""
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

func (u VersionedState) CurrentChartURL() string {
	return u.V1.ChartURL
}

func (v VersionedState) Versioned() VersionedState {
	return v
}
