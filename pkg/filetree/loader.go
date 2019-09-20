package filetree

import (
	"os"
	"path"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

const (
	CustomResourceDefinition = "CustomResourceDefinition"
	PatchesFolder            = "overlays"
	ResourcesFolder          = "resources"
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
	Logger        log.Logger
	FS            afero.Afero
	StateManager  state.Manager
	excludedBases map[string]string
	patches       map[string]string
	resources     map[string]string
}

func (a *aferoLoader) loadShipOverlay() error {
	currentState, err := a.StateManager.CachedState()
	if err != nil {
		return errors.Wrap(err, "failed to load state")
	}

	kustomize := currentState.CurrentKustomize()
	if kustomize == nil {
		kustomize = &state.Kustomize{}
	}

	shipOverlay := kustomize.Ship()
	baseMap := make(map[string]string)
	for _, base := range shipOverlay.ExcludedBases {
		baseMap[base] = base
	}
	a.excludedBases = baseMap
	a.patches = shipOverlay.Patches
	a.resources = shipOverlay.Resources
	return nil
}

func (a *aferoLoader) LoadTree(root string) (*Node, error) {
	if err := a.loadShipOverlay(); err != nil {
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
	patchesRootNode := Node{
		Path:     "/",
		Name:     PatchesFolder,
		Children: []Node{},
	}
	resourceRootNode := Node{
		Path:     "/",
		Name:     ResourcesFolder,
		Children: []Node{},
	}

	populatedBase, err := a.loadTree(fs, rootNode, files)
	if err != nil {
		return nil, errors.Wrap(err, "load tree")
	}

	populatedPatches := a.loadOverlayTree(patchesRootNode, a.patches)
	populatedResources := a.loadOverlayTree(resourceRootNode, a.resources)

	children := []Node{populatedBase}

	if len(populatedPatches.Children) != 0 {
		children = append(children, populatedPatches)
	}

	if len(populatedResources.Children) != 0 {
		children = append(children, populatedResources)
	}

	return &Node{
		Path:     "/",
		Name:     "/",
		Children: children,
	}, nil
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

		_, exists := a.excludedBases[filePath]
		return a.loadTree(fs, current.withChild(Node{
			Name:        file.Name(),
			Path:        filePath,
			HasOverlay:  hasOverlay,
			IsSupported: IsSupported(fileB),
			IsExcluded:  exists,
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

func IsSupported(file []byte) bool {
	resourceFactory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())

	resources, err := resourceFactory.SliceFromBytes(file)
	if err != nil {
		return false
	}
	if len(resources) != 1 {
		return false
	}
	r := resources[0]

	// any kind but CRDs are supported
	return r.GetKind() != CustomResourceDefinition
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

func (a *aferoLoader) loadOverlayTree(kustomizationNode Node, files map[string]string) Node {
	filledTree := &kustomizationNode
	for patchPath := range files {
		splitPatchPath := strings.Split(patchPath, "/")[1:]
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
