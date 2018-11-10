package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"gopkg.in/yaml.v2"
)

func (r *Resolver) DefaultHelmRelease(chartPath string) api.Spec {
	spec := api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{
				{
					Helm: &api.HelmAsset{
						AssetShared: api.AssetShared{
							Dest: constants.KustomizeBasePath,
						},
						Local: &api.LocalHelmOpts{
							ChartRoot: chartPath,
						},
						ValuesFrom: &api.ValuesFrom{
							Lifecycle: &api.ValuesFromLifecycle{},
						},
					},
				},
			},
		},
		Lifecycle: api.Lifecycle{
			V1: []api.Step{
				{
					HelmIntro: &api.HelmIntro{
						IsUpdate: r.Viper.GetBool("IsUpdate"),
						StepShared: api.StepShared{
							ID: "intro",
						},
					},
				},
				{
					HelmValues: &api.HelmValues{
						StepShared: api.StepShared{
							ID:          "values",
							Requires:    []string{"intro"},
							Invalidates: []string{"render"},
						},
					},
				},
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID:       "render",
							Requires: []string{"values"},
						},
						Root: ".",
					},
				},
				{
					KustomizeIntro: &api.KustomizeIntro{
						StepShared: api.StepShared{
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						Base:    constants.KustomizeBasePath,
						Overlay: path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:       "kustomize",
							Requires: []string{"render"},
						},
						Dest: "rendered.yaml",
					},
				},
			},
		},
	}
	if !r.NoOutro {
		spec.Lifecycle.V1 = append(spec.Lifecycle.V1, api.Step{
			Message: &api.Message{
				StepShared: api.StepShared{
					ID:       "outro",
					Requires: []string{"kustomize"},
				},
				Contents: `
Assets are ready to deploy. You can run

    kubectl apply -f rendered.yaml

to deploy the overlaid assets to your cluster.
`},
		})
	}

	return spec
}

func (r *Resolver) DefaultRawRelease(basePath string) api.Spec {
	spec := api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{},
		},
		Config: api.Config{
			V1: []libyaml.ConfigGroup{},
		},
		Lifecycle: api.Lifecycle{
			V1: []api.Step{
				{
					KustomizeIntro: &api.KustomizeIntro{
						StepShared: api.StepShared{
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						Base:    basePath,
						Overlay: path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:          "kustomize",
							Invalidates: []string{"diff"},
						},
						Dest: "rendered.yaml",
					},
				},
			},
		},
	}
	if !r.NoOutro {
		spec.Lifecycle.V1 = append(spec.Lifecycle.V1, api.Step{
			Message: &api.Message{
				StepShared: api.StepShared{
					ID:       "outro",
					Requires: []string{"kustomize"},
				},
				Contents: `
Assets are ready to deploy. You can run

    kubectl apply -f rendered.yaml

to deploy the overlaid assets to your cluster.
`},
		})
	}
	return spec
}

func (r *Resolver) resolveMetadata(ctx context.Context, upstream, localPath string, applicationType string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveHelmMetadata"))

	baseMetadata, err := r.ResolveBaseMetadata(upstream, localPath)
	if err != nil {
		return nil, errors.Wrap(err, "resolve base metadata")
	}

	if util.IsGithubURL(upstream) {
		releaseNotes, err := r.GitHubReleaseNotesFetcher.ResolveReleaseNotes(ctx, upstream)
		if err != nil {
			debug.Log("event", "releaseNotes.resolve.fail", "upstream", upstream, "err", err)
		}
		baseMetadata.ReleaseNotes = releaseNotes
	}

	err = r.StateManager.SerializeContentSHA(baseMetadata.ContentSHA)
	if err != nil {
		return nil, errors.Wrap(err, "write content sha")
	}

	localChartPath := filepath.Join(localPath, "Chart.yaml")

	exists, err := r.FS.Exists(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}
	if !exists {
		return baseMetadata, nil
	}

	debug.Log("phase", "read-chart", "from", localChartPath)
	chart, err := r.FS.ReadFile(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}

	debug.Log("phase", "unmarshal-chart.yaml")
	if err := yaml.Unmarshal(chart, &baseMetadata); err != nil {
		return nil, err
	}

	if err := r.StateManager.SerializeShipMetadata(*baseMetadata, applicationType); err != nil {
		return nil, errors.Wrap(err, "write metadata to state")
	}

	return baseMetadata, nil
}

// ResolveBaseMetadata resolves URL, ContentSHA, and Readme for the resource
func (r *Resolver) ResolveBaseMetadata(upstream string, localPath string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "resolveBaseMetaData"))
	var md api.ShipAppMetadata
	md.URL = upstream
	debug.Log("phase", "calculate-sha", "for", localPath)
	contentSHA, err := r.shaSummer(r, localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "calculate chart sha")
	}
	md.ContentSHA = contentSHA

	localReadmePath := filepath.Join(localPath, "README.md")
	debug.Log("phase", "read-readme", "from", localReadmePath)
	readme, err := r.FS.ReadFile(localReadmePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "read file from %s", localReadmePath)
		}
	}
	if readme != nil {
		md.Readme = string(readme)
	} else {
		// TODO default README better
		md.Readme = fmt.Sprintf(`
Deployment Generator
===========================

This is a deployment generator for

    ship init %s

Sources for your app have been generated at %s. This installer will walk you through
customizing these resources and preparing them to deploy to your infrastructure.
`, upstream, localPath)
	}
	return &md, nil
}

func (r *Resolver) maybeGetShipYAML(ctx context.Context, localPath string) (*api.Spec, error) {
	localReleasePaths := []string{
		filepath.Join(localPath, "ship.yaml"),
		filepath.Join(localPath, "ship.yml"),
	}

	r.ui.Info("Looking for ship.yaml ...")

	for _, shipYAMLPath := range localReleasePaths {
		upstreamShipYAMLExists, err := r.FS.Exists(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "check file %s exists", shipYAMLPath)
		}

		if !upstreamShipYAMLExists {
			continue
		}
		upstreamRelease, err := r.FS.ReadFile(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "read file from %s", shipYAMLPath)
		}
		var spec api.Spec
		if err := yaml.UnmarshalStrict(upstreamRelease, &spec); err != nil {
			level.Debug(r.Logger).Log("event", "release.unmarshal.fail", "error", err)
			return nil, errors.Wrapf(err, "unmarshal ship.yaml")
		}
		return &spec, nil
	}

	return nil, nil
}

type shaSummer func(r *Resolver, localPath string) (string, error)

func (r *Resolver) calculateContentSHA(root string) (string, error) {
	var contents []byte
	err := r.FS.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "fs walk")
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fileContents, err := r.FS.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read file")
		}

		contents = append(contents, fileContents...)
		return nil
	})

	if err != nil {
		return "", errors.Wrapf(err, "calculate content sha")
	}

	return fmt.Sprintf("%x", sha256.Sum256(contents)), nil
}

// this function is not perfect, and has known limitations. One of these is that it does not account for `\n---\n` in multiline strings.
func (r *Resolver) maybeSplitMultidocYaml(ctx context.Context, localPath string) error {
	type outputYaml struct {
		name     string
		contents string
	}

	files, err := r.FS.ReadDir(localPath)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", localPath)
	}

	if len(files) != 1 {
		// if there's more than one file, we'll assume that it does not need to be split
		// if there are no files, there's nothing to do
		return nil
	}

	file := files[0]

	if file.IsDir() {
		// if the single file is a directory, obviously we can't split it
		return nil
	}

	if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
		// not yaml, nothing to do
		return nil
	}

	inFileBytes, err := r.FS.ReadFile(filepath.Join(localPath, file.Name()))
	if err != nil {
		return errors.Wrapf(err, "read %s", filepath.Join(localPath, file.Name()))
	}

	outputFiles := []outputYaml{}
	filesStrings := strings.Split(string(inFileBytes), "\n---\n")

	// generate replacement yaml files
	for idx, fileString := range filesStrings {

		thisOutputFile := outputYaml{contents: fileString}

		thisMetadata := state.MinimalK8sYaml{}
		_ = yaml.Unmarshal([]byte(fileString), &thisMetadata)

		if thisMetadata.Kind == "" {
			// not a valid k8s yaml
			continue
		}

		fileName := generateNameFromMetadata(thisMetadata, idx)
		thisOutputFile.name = fileName
		outputFiles = append(outputFiles, thisOutputFile)
	}

	if len(outputFiles) < 2 {
		// not a multidoc yaml, or at least not a multidoc kubernetes yaml
		return nil
	}

	// delete multidoc yaml file
	err = r.FS.Remove(filepath.Join(localPath, file.Name()))
	if err != nil {
		return errors.Wrapf(err, "unable to remove %s", filepath.Join(localPath, file.Name()))
	}

	// write replacement yaml
	for _, outputFile := range outputFiles {
		err = r.FS.WriteFile(filepath.Join(localPath, outputFile.name+".yaml"), []byte(outputFile.contents), os.FileMode(0644))
		if err != nil {
			return errors.Wrapf(err, "write %s", outputFile.name)
		}
	}

	return nil
}

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func (r *Resolver) maybeSplitListYaml(ctx context.Context, path string) error {
	files, err := r.FS.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", path)
	}

	var lists []state.List
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())

		if file.IsDir() {
			continue
			// TODO: handling nested list yamls
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			return nil
		}

		fileB, err := r.FS.ReadFile(filePath)
		if err != nil {
			return errors.Wrapf(err, "read %s", filePath)
		}

		k8sYaml := ListK8sYaml{}
		if err := yaml.Unmarshal(fileB, &k8sYaml); err != nil {
			return errors.Wrapf(err, "unmarshal %s", filePath)
		}

		if k8sYaml.Kind == "List" {
			listItems := make([]state.MinimalK8sYaml, 0)
			for idx, item := range k8sYaml.Items {
				itemK8sYaml := state.MinimalK8sYaml{}
				itemB, err := yaml.Marshal(item)
				if err != nil {
					return errors.Wrapf(err, "marshal item %d from %s", idx, filePath)
				}

				if err := yaml.Unmarshal(itemB, &itemK8sYaml); err != nil {
					return errors.Wrap(err, "unmarshal item")
				}

				fileName := generateNameFromMetadata(itemK8sYaml, idx)
				if err := r.FS.WriteFile(filepath.Join(path, fileName+".yaml"), []byte(itemB), os.FileMode(0644)); err != nil {
					return errors.Wrap(err, "write yaml")
				}

				listItems = append(listItems, itemK8sYaml)
			}
			list := state.List{
				APIVersion: k8sYaml.APIVersion,
				Path:       filePath,
				Items:      listItems,
			}
			lists = append(lists, list)

			if err := r.FS.Remove(filePath); err != nil {
				return errors.Wrapf(err, "remove k8s list %s", filePath)
			}
		}
	}

	if err := r.StateManager.SerializeListsMetadata(lists); err != nil {
		return errors.Wrapf(err, "serialize list metadata")
	}

	return nil
}

func generateNameFromMetadata(k8sYaml state.MinimalK8sYaml, idx int) string {
	fileName := fmt.Sprintf("%s-%d", k8sYaml.Kind, idx)

	if k8sYaml.Metadata.Name != "" {
		fileName = k8sYaml.Kind + "-" + k8sYaml.Metadata.Name
		if k8sYaml.Metadata.Namespace != "" && k8sYaml.Metadata.Namespace != "default" {
			fileName += "-" + k8sYaml.Metadata.Namespace
		}
	}

	return fileName
}
