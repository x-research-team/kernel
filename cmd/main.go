package main

import (
	"github.com/x-research-team/bus"
	"github.com/x-research-team/bus/pipe"
	"github.com/x-research-team/implant"
	"github.com/x-research-team/kernel/internal/kernel"
	"github.com/x-research-team/kernel/pkg/sys"
	"github.com/x-research-team/vm"
)

func init() {
	// Enable system logging
	sys.Trace(true)

	// Initialize core kernel parts
	bus.Init("kernel/config/log.json")
	pipe.Init()
	vm.Init()
	implant.Init(
		"external/system",
		"external/services",
		"external/security",
		"external/applications",
	)
}

func main() {
	if err := kernel.New(implant.Modules()...).Run(); err != nil {
		bus.Error <- err
	}
}
