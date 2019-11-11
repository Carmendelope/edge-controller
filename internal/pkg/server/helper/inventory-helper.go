/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package helper

import (
	"fmt"
	"github.com/nalej/derrors"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/sysinfo"
	"github.com/rs/zerolog/log"
	"go/build"
)

var osToClass = map[string]grpc_inventory_go.OperatingSystemClass{
	"linux": grpc_inventory_go.OperatingSystemClass_LINUX,
	"windows": grpc_inventory_go.OperatingSystemClass_WINDOWS,
	"darwin": grpc_inventory_go.OperatingSystemClass_DARWIN,
}

type Inventory struct {
	*sysinfo.SysInfo
}


func NewInventory() (*Inventory, derrors.Error) {
	log.Debug().Msg("gathering system inventory information")

	i := &Inventory{sysinfo.NewSysInfo()}
	log.Debug().Interface("sysinfo", i).Msg("inventory information")

	return i, nil
}

func (i *Inventory) GetOSClass() grpc_inventory_go.OperatingSystemClass {
	return osToClass[build.Default.GOOS]
}

func (i *Inventory) GetAssetInfo() (*grpc_inventory_go.AssetInfo) {

	netInfo := make([]*grpc_inventory_go.NetworkingHardwareInfo, 0, len(i.Network))
	for _, iface := range i.Network  {
		grpcIface := &grpc_inventory_go.NetworkingHardwareInfo{
			Type: iface.Port,
			LinkCapacity: int64(iface.Speed),
		}
		netInfo = append(netInfo, grpcIface)
	}

	cpuInfo := make([]*grpc_inventory_go.CPUInfo, 0)
	if i.CPU != nil {
		for count := uint(0); count < i.CPU.Cpus; count++ {
			cpuInfo = append(cpuInfo, &grpc_inventory_go.CPUInfo{
				Manufacturer: i.CPU.Vendor,
				Model: i.CPU.Model,
				NumCores: int32(i.CPU.Cores),
			})
		}
	}

	storageInfo := make([]*grpc_inventory_go.StorageHardwareInfo, 0, len(i.Storage))
	for _, s := range i.Storage {
		grpcStorage := &grpc_inventory_go.StorageHardwareInfo{
			Type: fmt.Sprintf("%s %s", s.Vendor, s.Model),
			TotalCapacity: int64(s.Size),
		}
		storageInfo = append(storageInfo, grpcStorage)
	}

	os := &grpc_inventory_go.OperatingSystemInfo{
		Class: i.GetOSClass(),
	}
	if i.OS != nil {
		os.Name = i.OS.Name
		os.Architecture = i.OS.Architecture
	}
	if i.Kernel != nil {
		os.Version = i.Kernel.Release
	}

	hardware := &grpc_inventory_go.HardwareInfo{
		Cpus: cpuInfo,
		NetInterfaces: netInfo,
	}
	if i.Memory != nil {
		hardware.InstalledRam = int64(i.Memory.Size)
	}

	assetInfo := &grpc_inventory_go.AssetInfo{
		Os: os,
		Hardware: hardware,
		Storage: storageInfo,
	}

	return assetInfo
}
