package server

import (
	"github.com/x-research-team/kernel/external/system/server/component"

	"github.com/x-research-team/contract"
)

// Init Load plugin with all components
func Init() contract.KernelModule {
	return component.New(
		component.Configure(),
	)
}
