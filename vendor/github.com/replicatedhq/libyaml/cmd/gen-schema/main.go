// package main provides an entrypoint for generating a JsonSchema document from the definitions
// in the libyaml package
package main

import (
	"fmt"
	"github.com/urakozz/go-json-schema-generator"
	"github.com/replicatedhq/libyaml"

)

func main(){
	fmt.Println(generator.Generate(&libyaml.RootConfig{}))
}
