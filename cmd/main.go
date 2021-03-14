package main

import (
	"github.com/x-research-team/bus"
	"github.com/x-research-team/bus/pipe"
	"github.com/x-research-team/implant"
	"github.com/x-research-team/kernel/external/system/server"
	"github.com/x-research-team/kernel/external/system/storage"
	"github.com/x-research-team/kernel/internal/config"
	"github.com/x-research-team/kernel/internal/kernel"
	"github.com/x-research-team/kernel/internal/sys"
	"github.com/x-research-team/vm"
)

func init() {
	// Enable system logging
	sys.Trace(true)

	// Initialize core kernel parts
	logger := config.Kernel.Log.Level.ToJson()
	bus.Init(logger)
	pipe.Init()
	vm.Init()

	components := config.Kernel.Components.Paths()
	implant.Init(components...)
}

func main() {
	modules := implant.Modules()
	modules = append(modules, storage.Init(), server.Init())
	if err := kernel.New(modules...).Run(); err != nil {
		bus.Error <- err
	}
}
