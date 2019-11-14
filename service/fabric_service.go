// Copyright 2019 VMware, Inc. All rights reserved. -- VMware Confidential

package service

import (
    "go-bifrost/model"
)

// Interface containing all APIs which should be implemented by Fabric Services.
type FabricService interface {
    // Handles a single Fabric Request
    HandleServiceRequest(request *model.Request, core FabricServiceCore)
}

// Optional interface, if implemented by a fabric service, its Init method
// will be invoked when the service is registered in the ServiceRegistry.
type FabricInitializableService interface {
    Init(core FabricServiceCore) error
}

