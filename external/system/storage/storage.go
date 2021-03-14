package storage

import (
	"github.com/x-research-team/contract"
	"github.com/x-research-team/kernel/external/system/storage/component"
	"github.com/x-research-team/kernel/external/system/storage/component/dsn"
)

// Init Load plugin with all components
func Init() contract.KernelModule {
	return component.New(
		component.ConnectTo(dsn.Parse()),
	)
}
