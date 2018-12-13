package main

import (
	"fmt"

	jsonschema "github.com/mcuadros/go-jsonschema-generator"
	"github.com/replicatedhq/ship/pkg/api"
)

func main() {
	s := &jsonschema.Document{}
	s.Read(&api.Spec{})
	fmt.Println(s)
}
