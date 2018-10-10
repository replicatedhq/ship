package filetree

import (
	"bytes"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/kustomize/pkg/resource"
)

const (
	CustomResourceDefinition = "CustomResourceDefinition"
	OverlaysFolder           = "overlays"
)

// A Loader returns a struct representation
// of a filesystem directory tree
type Loader interface {
	LoadTree(root string) (*Node, error)
	// someday this should return an overlay too
	LoadFile(root string, path string) ([]byte, error)
}

// NewLoader builds an aferoLoader, used with dig
func NewLoader(
	fs afero.Afero,
	logger log.Logger,
	stateManager state.Manager,
) Loader {
	return &aferoLoader{
		FS:           fs,
		Logger:       logger,
		StateManager: stateManager,
	}
}

type aferoLoader struct {
	Logger       log.Logger
	FS           afero.Afero
	StateManager state.Manager
	patches      map[string]string
	resources    map[string]string
}

func (a *aferoLoader) loadShipPatches() error {
	currentState, err := a.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "failed to load state")
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	shipOverlay := kustomize.Ship()
	a.patches = shipOverlay.Patches
	return nil
}

func (a *aferoLoader) loadShipResources() error {
	currentState, err := a.StateManager.TryLoad()
	if err != nil {
		return errors.Wrap(err, "failed to load state")
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	shipOverlay := kustomize.Ship()
	a.resources = shipOverlay.Resources
	return nil
}

func (a *aferoLoader) LoadTree(root string) (*Node, error) {
	if err := a.loadShipPatches(); err != nil {
		return nil, errors.Wrapf(err, "load overlays")
	}

	fs := afero.Afero{Fs: afero.NewBasePathFs(a.FS, root)}

	files, err := fs.ReadDir("/")
	if err != nil {
		return nil, errors.Wrapf(err, "read dir %q", root)
	}

	rootNode := Node{
		Path:     "/",
		Name:     "/",
		Children: []Node{},
	}
	overlayRootNode := Node{
		Path:     "/",
		Name:     OverlaysFolder,
		Children: []Node{},
	}

	populatedKustomization := a.loadOverlayTree(overlayRootNode)
	populated, err := a.loadTree(fs, rootNode, files)
	children := []Node{populated}

	if len(populatedKustomization.Children) != 0 {
		children = append(children, populatedKustomization)
	}

	return &Node{
		Path:     "/",
		Name:     "/",
		Children: children,
	}, errors.Wrap(err, "load tree")
}

// todo move this to a new struct or something
func (a *aferoLoader) LoadFile(root string, file string) ([]byte, error) {
	fs := afero.Afero{Fs: afero.NewBasePathFs(a.FS, root)}
	contents, err := fs.ReadFile(file)
	if err != nil {
		return []byte{}, errors.Wrap(err, "read file")
	}

	return contents, nil
}

func (a *aferoLoader) loadTree(fs afero.Afero, current Node, files []os.FileInfo) (Node, error) {
	if len(files) == 0 {
		return current, nil
	}

	file, rest := files[0], files[1:]
	filePath := path.Join(current.Path, file.Name())

	// no thanks
	if isSymlink(file) {
		level.Debug(a.Logger).Log("event", "symlink.skip", "file", filePath)
		return a.loadTree(fs, current, rest)
	}

	if !file.IsDir() {
		_, hasOverlay := a.patches[filePath]

		fileB, err := fs.ReadFile(filePath)
		if err != nil {
			return current, errors.Wrapf(err, "read file %s", file.Name())
		}

		return a.loadTree(fs, current.withChild(Node{
			Name:        file.Name(),
			Path:        filePath,
			HasOverlay:  hasOverlay,
			IsSupported: isSupported(fileB),
		}), rest)
	}

	subFiles, err := fs.ReadDir(filePath)
	if err != nil {
		return current, errors.Wrapf(err, "read dir %q", file.Name())
	}

	subTree := Node{
		Name:     file.Name(),
		Path:     filePath,
		Children: []Node{},
	}

	subTreeLoaded, err := a.loadTree(fs, subTree, subFiles)
	if err != nil {
		return current, errors.Wrapf(err, "load tree %q", file.Name())
	}

	return a.loadTree(fs, current.withChild(subTreeLoaded), rest)
}

func isSymlink(file os.FileInfo) bool {
	return file.Mode()&os.ModeSymlink != 0
}

func isSupported(file []byte) bool {
	var out unstructured.Unstructured

	fileJSON, err := yaml.YAMLToJSON(file)
	if err != nil {
		return false
	}

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(fileJSON), 1024)
	if err := decoder.Decode(&out); err != nil {
		return false
	}

	r := resource.NewResourceFromUnstruct(out)
	if r.GetKind() == CustomResourceDefinition {
		return false
	}

	return true
}

func (n Node) withChild(child Node) Node {
	return Node{
		Name:        n.Name,
		Path:        n.Path,
		Children:    append(n.Children, child),
		IsSupported: n.IsSupported,
		HasOverlay:  n.HasOverlay,
	}
}

func (a *aferoLoader) loadOverlayTree(kustomizationNode Node) Node {
	filledTree := &kustomizationNode
	for patchPath := range a.patches {
		splitPatchPath := strings.Split(patchPath, "/")[1:]
		filledTree = a.createOverlayNode(filledTree, splitPatchPath)
	}
	for resourcePath := range a.resources {
		splitPatchPath := strings.Split(resourcePath, "/")[1:]
		filledTree = a.createOverlayNode(filledTree, splitPatchPath)
	}
	return *filledTree
}

func (a *aferoLoader) createOverlayNode(kustomizationNode *Node, pathToOverlay []string) *Node {
	if len(pathToOverlay) == 0 {
		return kustomizationNode
	}

	pathToMatch, restOfPath := pathToOverlay[0], pathToOverlay[1:]
	filePath := path.Join(kustomizationNode.Path, pathToMatch)

	for i := range kustomizationNode.Children {
		if kustomizationNode.Children[i].Path == pathToMatch || kustomizationNode.Children[i].Name == pathToMatch {
			a.createOverlayNode(&kustomizationNode.Children[i], restOfPath)
			return kustomizationNode
		}
	}

	nextNode := Node{
		Name: pathToMatch,
		Path: filePath,
	}
	loadedChild := a.createOverlayNode(&nextNode, restOfPath)
	kustomizationNode.Children = append(kustomizationNode.Children, *loadedChild)
	return kustomizationNode
}
