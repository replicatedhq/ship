package resolve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/templates"
)

type depGraph struct {
	BuilderBuilder *templates.BuilderBuilder
	Dependencies   map[string]map[string]struct{}
}

//these config functions are used to add their dependencies to the depGraph
func (d *depGraph) funcMap(parent string) template.FuncMap {
	addDepFunc := func(dep string, _ ...string) string {
		d.AddDep(parent, dep)
		return dep
	}

	return template.FuncMap{
		"ConfigOption":          addDepFunc,
		"ConfigOptionIndex":     addDepFunc,
		"ConfigOptionData":      addDepFunc,
		"ConfigOptionEquals":    addDepFunc,
		"ConfigOptionNotEquals": addDepFunc,
	}
}

func (d *depGraph) AddNode(source string) {
	if d.Dependencies == nil {
		d.Dependencies = make(map[string]map[string]struct{})
	}

	if _, ok := d.Dependencies[source]; !ok {
		d.Dependencies[source] = make(map[string]struct{})
	}
}

func (d *depGraph) AddDep(source, newDependency string) {
	d.AddNode(source)

	d.Dependencies[source][newDependency] = struct{}{}
}

func (d *depGraph) ResolveDep(resolvedDependency string) {
	for _, depMap := range d.Dependencies {
		delete(depMap, resolvedDependency)
	}
	delete(d.Dependencies, resolvedDependency)
}

func (d *depGraph) GetHeadNodes() ([]string, error) {
	headNodes := []string{}

	for node, deps := range d.Dependencies {
		if len(deps) == 0 {
			headNodes = append(headNodes, node)
		}
	}

	if len(headNodes) == 0 && len(d.Dependencies) != 0 {
		return headNodes, errors.New("No nodes exist with 0 dependencies")
	}

	return headNodes, nil
}

func (d *depGraph) PrintData() string {
	return fmt.Sprintf("deps: %+v", d.Dependencies)
}

// returns a deep copy of the dep graph
func (d *depGraph) Copy() (depGraph, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	dec := json.NewDecoder(&buf)
	err := enc.Encode(d.Dependencies)
	if err != nil {
		return depGraph{}, err
	}
	var copy map[string]map[string]struct{}
	err = dec.Decode(&copy)
	if err != nil {
		return depGraph{}, err
	}

	return depGraph{
		BuilderBuilder: d.BuilderBuilder,
		Dependencies:   copy,
	}, nil

}

func (d *depGraph) ParseConfigGroup(configGroups []libyaml.ConfigGroup) error {
	staticCtx := d.BuilderBuilder.NewStaticContext()
	for _, configGroup := range configGroups {
		for _, configItem := range configGroup.Items {
			// add this to the dependency graph
			d.AddNode(configItem.Name)

			depBuilder := d.BuilderBuilder.NewBuilder(staticCtx)
			depBuilder.Functs = d.funcMap(configItem.Name)

			// while builder is normally stateless, the functions it uses within this loop are not
			// errors are also discarded as we do not have the full set of template functions available here
			_, _ = depBuilder.String(configItem.Default)
			_, _ = depBuilder.String(configItem.Value)
		}
	}

	return nil
}
