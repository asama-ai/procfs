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
// +build linux

package sysfs

import (
	"path/filepath"
)

// Note: All AER types (CorrectableAerCounters, UncorrectableAerCounters, AerCounters, AllAerCounters)
// and the parsing functions (ParseAerCounters, parseCorrectableAerCounters, parseUncorrectableAerCounters)
// are defined in pci_device.go (same package, so accessible here). This file maintains the public API
// methods for backward compatibility and delegates to the shared implementation in pci_device.go.

// AerCountersByIface returns info for a single net interfaces (iface).
func (fs FS) AerCountersByIface(devicePath string) (*AerCounters, error) {
	_, err := fs.NetClassByIface(devicePath)
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	counters, err := parseAerCounters(filepath.Join(path, devicePath))
	if err != nil {
		return nil, err
	}
	counters.Name = devicePath

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
		counters, err := parseAerCounters(filepath.Join(path, devicePath))
		if err != nil {
			return nil, err
		}
		counters.Name = devicePath
		allAerCounters[devicePath] = *counters
	}

	return allAerCounters, nil
}
