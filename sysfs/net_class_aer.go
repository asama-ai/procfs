// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package sysfs

import (
	"path/filepath"
)

type AerCounters struct {
	PciDeviceAerCounters
	Name string // Interface name
}

// Note: The generic AER types (CorrectableAerCounters, UncorrectableAerCounters, PciDeviceAerCounters)
// and the parsing functions (parseAerCounters, parseCorrectableAerCounters, parseUncorrectableAerCounters)
// are defined in pci_device.go
// This file maintains the public API methods for backward compatibility and delegates to the shared implementation in pci_device.go.

// AllAerCounters is collection of AER counters for every interface (iface) in /sys/class/net.
// The map keys are interface (iface) names.
type AllAerCounters map[string]AerCounters

// AerCountersByIface returns info for a single net interfaces (iface).
func (fs FS) AerCountersByIface(devicePath string) (*AerCounters, error) {
	_, err := fs.NetClassByIface(devicePath)
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	Counters, err := parseAerCounters(filepath.Join(path, devicePath))
	if err != nil {
		return nil, err
	}
	if Counters == nil {
		// AER not supported for this device
		return nil, nil
	}

	// Convert PciDeviceAerCounters to AerCounters by embedding and adding Name
	counters := &AerCounters{
		PciDeviceAerCounters: *Counters,
		Name:                 devicePath,
	}

	return counters, nil
}

// AerCounters returns AER counters for all net interfaces (iface) read from /sys/class/net/<iface>/device.
func (fs FS) AerCounters() (AllAerCounters, error) {
	devices, err := fs.NetClassDevices()
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	allAerCounters := AllAerCounters{}
	for _, devicePath := range devices {
		pciCounters, err := parseAerCounters(filepath.Join(path, devicePath, "device"))
		if err != nil {
			return nil, err
		}
		counters := AerCounters{
			PciDeviceAerCounters: *pciCounters,
			Name:                 devicePath,
		}
		allAerCounters[devicePath] = counters
	}

	return allAerCounters, nil
}
