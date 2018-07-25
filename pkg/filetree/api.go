package filetree

type Node struct {
	Children   []Node `json:"children" yaml:"children"`
	Name       string `json:"name" yaml:"name"`
	Path       string `json:"path" yaml:"path"`
	HasOverlay bool   `json:"hasOverlay,omitempty" yaml:"hasOverlay,omitempty"`
}
