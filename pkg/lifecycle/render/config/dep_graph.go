package config

import (
	"github.com/pkg/errors"
)

type depGraph struct {
	dependencies map[string]map[string]struct{}
}

func (d *depGraph) addNode(source string) {
	if d.dependencies == nil {
		d.dependencies = make(map[string]map[string]struct{})
	}

	if _, ok := d.dependencies[source]; !ok {
		d.dependencies[source] = make(map[string]struct{})
	}
}

func (d *depGraph) addDep(source, newDependency string) {
	d.addNode(source)

	d.dependencies[source][newDependency] = struct{}{}
}

func (d *depGraph) resolveDep(resolvedDependency string) {
	for _, depMap := range d.dependencies {
		delete(depMap, resolvedDependency)
	}
	delete(d.dependencies, resolvedDependency)
}

func (d *depGraph) getHeadNodes() ([]string, error) {
	headNodes := []string{}

	for node, deps := range d.dependencies {
		if len(deps) == 0 {
			headNodes = append(headNodes, node)
		}
	}

	if len(headNodes) == 0 && len(d.dependencies) != 0 {
		return headNodes, errors.New("No nodes exist with 0 dependencies")
	}

	return headNodes, nil
}
