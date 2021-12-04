// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package service

import (
	"github.com/vmware/transport-go/model"
)

// FabricService Interface containing all APIs which should be implemented by Fabric Services.
type FabricService interface {
	// Handles a single Fabric Request
	HandleServiceRequest(request *model.Request, core FabricServiceCore)
}

// FabricInitializableService Optional interface, if implemented by a fabric service, its Init method
// will be invoked when the service is registered in the ServiceRegistry.
type FabricInitializableService interface {
	Init(core FabricServiceCore) error
}
