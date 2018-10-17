// Copyright © 2017 Naveego

package main

import (
	"github.com/naveego/plugin-oracle/cmd"
	_ "gopkg.in/goracle.v2"
	// imported to prevent dep from evicting it, dep doesn't scan magefile.go
	_ 	"github.com/naveego/ci/go/build"

)

func main() {
	cmd.Execute()
}
