package main

import (
	"github.com/x-research-team/bus"
	"github.com/x-research-team/bus/pipe"
	"github.com/x-research-team/implant"
	"github.com/x-research-team/kernel/internal/config"
	"github.com/x-research-team/kernel/internal/kernel"
	"github.com/x-research-team/kernel/internal/sys"
	"github.com/x-research-team/vm"
)

func init() {
	// Enable system logging
	sys.Trace(true)

	// Initialize core kernel parts
	bus.Init(config.Kernel.Log.Level.ToJson())
	pipe.Init()
	vm.Init()
	implant.Init(config.Kernel.Components.Paths()...)
}

func main() {
	if err := kernel.New(implant.Modules()...).Run(); err != nil {
		bus.Error <- err
	}
}
