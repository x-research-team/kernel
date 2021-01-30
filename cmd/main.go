package main

import (
	"github.com/x-research-team/bus"
	"github.com/x-research-team/bus/pipe"
	"github.com/x-research-team/implant"
	"github.com/x-research-team/kernel/pkg/sys"
	"github.com/x-research-team/vm"
)

func init() {
	bus.Init("config/log.json")
	pipe.Init()
	sys.Trace(true)
	vm.Init()
	implant.Init(
		"system",
		"services",
		"security",
		"applications",
	)
}

func main() {

}
