package config

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
)

type depGraph struct {
	Dependencies map[string]map[string]struct{}
}

//these config functions are used to add their dependencies to the depGraph
func (d *depGraph) FuncMap(parent string) template.FuncMap {
	addDepFunc := func(dep string) string {
		d.AddDep(parent, dep)
		return dep
	}
	addDepFunc2Vars := func(dep, irrelevant string) string {
		d.AddDep(parent, dep)
		return dep
	}

	return template.FuncMap{
		"ConfigOption":          addDepFunc,
		"ConfigOptionIndex":     addDepFunc,
		"ConfigOptionData":      addDepFunc,
		"ConfigOptionEquals":    addDepFunc2Vars,
		"ConfigOptionNotEquals": addDepFunc2Vars,
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
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(d)
	if err != nil {
		return depGraph{}, err
	}
	var copy depGraph
	err = dec.Decode(&copy)
	if err != nil {
		return depGraph{}, err
	}
	return copy, nil
}
