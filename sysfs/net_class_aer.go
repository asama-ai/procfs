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

// Note: The generic AER types (CorrectableAerCounters, UncorrectableAerCounters, PciDeviceAerCounters)
// and the parsing functions (parseAerCounters, parseCorrectableAerCounters, parseUncorrectableAerCounters)
// are moved to pci_device_aer.go
// The public types (AerCounters, AllAerCounters) and their behavior are retained as is in this file for backward compatibility.

// AerCounters contains AER counters from files in /sys/class/net/<iface>/device
// for single interface (iface).
type AerCounters struct {
	Name string // Interface name
	PciDeviceAerCounters
}

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
	counters, err := parseAerCounters(filepath.Join(path, devicePath, "device"))
	if err != nil {
		return nil, err
	}
	if counters == nil {
		// AER not supported for this device
		return nil, nil
	}

	// Convert PciDeviceAerCounters to AerCounters by embedding and adding Name
	return &AerCounters{
		PciDeviceAerCounters: *counters,
		Name:                 devicePath,
	}, nil
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
		counters, err := parseAerCounters(filepath.Join(path, devicePath, "device"))
		if err != nil {
			return nil, err
		}
		allAerCounters[devicePath] = AerCounters{
			Name:                 devicePath,
			PciDeviceAerCounters: *counters,
		}
	}

	return allAerCounters, nil
}
