// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
    "fmt"
    "github.com/google/uuid"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/model"
    "gitlab.eng.vmware.com/bifrost/go-bifrost/service"
    "math/rand"
    "reflect"
    "strconv"
    "sync"
)

type VmRef struct {
    VcGuid string `json:"vcGuid"`
    VmId   string `json:"vmId"`
}

const (
    powerState_poweredOff = "poweredOff"
    powerState_poweredOn = "poweredOn"
    powerState_suspended = "suspended"

    diskFormat_native_512 = "native_512"
    diskFormat_emulated_512 = "emulated_512"
    diskFormat_native_4k = "native_4k"

    powerOperation_powerOn = "powerOn"
    powerOperation_powerOff = "powerOff"
    powerOperation_reset = "reset"
    powerOperation_suspend = "suspend"
)

type RuntimeInfo struct {
    Host       string `json:"host"`
    PowerState string `json:"powerState"`
}

type VirtualDisk struct {
    Key int `json:"key"`
    DeviceType string `json:"deviceType"`
    DeviceName string `json:"deviceName"`
    CapacityMB int64 `json:"capacityMB"`
    DiskFormat string `json:"diskFormat"`
}

type VirtualUSB struct {
    Key int `json:"key"`
    DeviceType string `json:"deviceType"`
    DeviceName string `json:"deviceName"`
    Connected bool `json:"connected"`
    Speed []string `json:"speed"`
}

type VirtualHardware struct {
    MemoryMB int `json:"memoryMB"`
    NumCPU int `json:"numCPU"`
    Devices []interface{} `json:"devices"`
}

type VirtualMachine struct {
    VmRef       VmRef            `json:"vmRef"`
    Name        string           `json:"name"`
    RuntimeInfo *RuntimeInfo     `json:"runtimeInfo"`
    Hardware    *VirtualHardware `json:"hardware"`
}

type VmListResponse struct {
    Error bool `json:"error"`
    ErrorMessage string `json:"errorMessage"`
    VirtualMachines []VirtualMachine `json:"virtualMachines"`
}

type VmCreateResponse struct {
    Error bool `json:"error"`
    ErrorMessage string `json:"errorMessage"`
    Vm VirtualMachine `json:"vm"`
}

type BaseVmResponse struct {
    Error bool `json:"error"`
    ErrorMessage string `json:"errorMessage"`
}

type VmPowerOperationResponseItem struct {
    VmRef VmRef `json:"vmRef"`
    OperationResult bool `json:"operationResult"`
}

type VmPowerOperationResponse struct {
    Error bool `json:"error"`
    ErrorMessage string `json:"errorMessage"`
    OpResults []VmPowerOperationResponseItem `json:"opResults"`
}

type VmDeleteRequest struct {
    Vm VmRef `json:"vm"`
}

type VmPowerOperationRequest struct {
    VmRefs []VmRef `json:"vmRefs"`
    PowerOperation string `json:"powerOperation"`
}

type VmCreateRequest struct {
    Name string `json:"name"`
    VirtualHardware *VirtualHardware `json:"virtualHardware"`
}

type vmService struct {
    vms []VirtualMachine
    lock sync.Mutex
}

func (s *vmService) Init(core service.FabricServiceCore) error {

    s.vms = []VirtualMachine {
        createVm("sample-go-vm1", &VirtualHardware{
            MemoryMB: 2048,
            NumCPU:   4,
            Devices:  []interface{} {
                VirtualDisk{
                    Key:        1,
                    DeviceType: "VirtualDisk",
                    DeviceName: "disk-1",
                    CapacityMB: 50000,
                    DiskFormat: diskFormat_emulated_512,
                },
                VirtualDisk{
                    Key:        2,
                    DeviceType: "VirtualDisk",
                    DeviceName: "disk-2",
                    CapacityMB: 150000,
                    DiskFormat: diskFormat_native_4k,
                },
            },
        }),

        createVm("sample-go-vm2", &VirtualHardware{
            MemoryMB: 8192,
            NumCPU:   8,
            Devices:  []interface{} {
                VirtualDisk{
                    Key:        1,
                    DeviceType: "VirtualDisk",
                    DeviceName: "disk-1",
                    CapacityMB: 50000,
                    DiskFormat: diskFormat_emulated_512,
                },
                VirtualUSB{
                    Key:        2,
                    DeviceType: "VirtualUSB",
                    DeviceName: "disk-2",
                    Connected: true,
                    Speed: []string {"superSpeed", "high"},
                },
            },
        }),
    }

    return nil
}

func (s *vmService) HandleServiceRequest(
        request *model.Request, core service.FabricServiceCore) {

    switch request.Request {
    case "listVms":
        core.SendResponse(request, &VmListResponse{
            VirtualMachines: s.vms,
        })
    case "deleteVm":
        s.deleteVm(request, core)
    case "changeVmPowerState":
        s.changePowerState(request, core)
    case "createVm":
        core.SendResponse(request, s.createVm(request))
    default:
        core.HandleUnknownRequest(request)
    }
}

func (s *vmService) deleteVm(request *model.Request, core service.FabricServiceCore) {
    reqPayload, err := model.ConvertValueToType(request.Payload, reflect.TypeOf(&VmDeleteRequest{}))
    if err != nil || reqPayload == nil {
        core.SendResponse(request, &BaseVmResponse{
            Error:        true,
            ErrorMessage: "Request payload should be VmDeleteRequest!",
        })
        return
    }

    deleteReq := reqPayload.(*VmDeleteRequest)

    s.lock.Lock()
    defer s.lock.Unlock()

    for i, vm := range s.vms {
        if deleteReq.Vm == vm.VmRef {
            copy(s.vms[i:], s.vms[i + 1:])
            s.vms = s.vms[:len(s.vms) - 1]
            core.SendResponse(request, &BaseVmResponse{})
            return
        }
    }

    core.SendResponse(request, &BaseVmResponse{
        Error:        true,
        ErrorMessage: fmt.Sprint("Cannot find VM:", deleteReq.Vm),
    })
}

func (s *vmService) createVm(request *model.Request)  *VmCreateResponse {
    resp := &VmCreateResponse{}

    reqPayload, err := model.ConvertValueToType(request.Payload, reflect.TypeOf(&VmCreateRequest{}))
    if err != nil || reqPayload == nil {
        resp.Error = true
        resp.ErrorMessage = "Request payload should be VmCreateRequest!"
        return resp
    }

    createReq := reqPayload.(*VmCreateRequest)
    if createReq.Name == "" {
        resp.Error = true
        resp.ErrorMessage = "Invalid VmCreateRequest: null name!"
        return resp
    }

    if createReq.VirtualHardware == nil {
        resp.Error = true
        resp.ErrorMessage = "Invalid VmCreateRequest: null virtualHardware!"
        return resp
    }

    for i, device := range createReq.VirtualHardware.Devices {

        deviceAsMap, ok := device.(map[string]interface{})
        if !ok || deviceAsMap == nil {
            resp.Error = true
            resp.ErrorMessage = "Invalid VmCreateRequest: invalid device"
            return resp
        }

        var deviceType reflect.Type
        switch deviceAsMap["deviceType"] {
        case "VirtualDisk":
            deviceType = reflect.TypeOf(VirtualDisk{})
        case "VirtualUSB":
            deviceType = reflect.TypeOf(VirtualUSB{})
        default:
            resp.Error = true
            resp.ErrorMessage = "Invalid VmCreateRequest: unsupported device type "
            return resp
        }

        createReq.VirtualHardware.Devices[i], err = model.ConvertValueToType(deviceAsMap, deviceType)
        if err != nil {
            resp.Error = true
            resp.ErrorMessage = "Invalid VmCreateRequest: invalid device"
            return resp
        }
    }

    s.lock.Lock()
    defer s.lock.Unlock()

    vm := createVm(createReq.Name, createReq.VirtualHardware)
    s.vms = append(s.vms, vm)
    resp.Vm = vm
    return resp
}

func createVm(name string, hardware *VirtualHardware) VirtualMachine {
    return VirtualMachine{
        VmRef:       VmRef{
            VcGuid: uuid.New().String(),
            VmId:   uuid.New().String(),
        },
        Name:        name,
        RuntimeInfo: &RuntimeInfo{
            Host:       "191.168.12." + strconv.Itoa(rand.Intn(255)),
            PowerState: powerState_poweredOff,
        },
        Hardware:    hardware,
    }
}

func (s *vmService) changePowerState(request *model.Request, core service.FabricServiceCore) {
    reqPayload, err := model.ConvertValueToType(
            request.Payload, reflect.TypeOf(&VmPowerOperationRequest{}))
    if err != nil || reqPayload == nil {
        core.SendResponse(request, &BaseVmResponse{
            Error:        true,
            ErrorMessage: "Request payload should be VmPowerOperationRequest!",
        })
        return
    }

    s.lock.Lock()
    defer s.lock.Unlock()

    vmPowerOpReq := reqPayload.(*VmPowerOperationRequest)
    result := &VmPowerOperationResponse{}
    for _, vmRef := range vmPowerOpReq.VmRefs {
        result.OpResults = append(result.OpResults, VmPowerOperationResponseItem{
            VmRef:           vmRef,
            OperationResult: s.changePowerStateOfVm(vmRef, vmPowerOpReq.PowerOperation),
        })
    }
    core.SendResponse(request, result)
}

func (s *vmService) changePowerStateOfVm(ref VmRef, newPowerState string) bool {
    for _, vm := range s.vms {
        if vm.VmRef == ref {
            switch newPowerState {
            case powerOperation_powerOff:
                if vm.RuntimeInfo.PowerState == powerState_poweredOff {
                    // We cannot power off powered off VMs
                    return false
                }
                vm.RuntimeInfo.PowerState = powerState_poweredOff
                return true;

            case powerOperation_reset:
                if vm.RuntimeInfo.PowerState == powerState_poweredOff ||
                        vm.RuntimeInfo.PowerState == powerState_suspended {
                    return false
                }
                vm.RuntimeInfo.PowerState = powerState_poweredOn
                return true

            case powerOperation_powerOn:
                if vm.RuntimeInfo.PowerState == powerState_poweredOn {
                    // We cannot power on powered on VMs
                    return false
                }
                vm.RuntimeInfo.PowerState = powerState_poweredOn
                return true

            case powerOperation_suspend:
                if vm.RuntimeInfo.PowerState == powerState_poweredOff ||
                        vm.RuntimeInfo.PowerState == powerState_suspended {
                    return false
                }
                vm.RuntimeInfo.PowerState = powerState_suspended
                return true
            default:
                return false
            }
        }
    }
    return false
}
