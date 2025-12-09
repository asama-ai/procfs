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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

const pcieportDriverPath = "bus/pci/drivers/pcieport"

type RootPortAerCounters struct {
	TotalErrCor      uint64
	TotalErrFatal    uint64
	TotalErrNonFatal uint64
}

// AllRootPortAerCounters is collection of root port AER counters for every root port device
// in /sys/bus/pci/drivers/pcieport.
// The map keys are device names (e.g., "0000:00:02.1").
type AllRootPortAerCounters map[string]RootPortAerCounters

// RootPortDevices scans /sys/bus/pci/drivers/pcieport for devices and returns them as a list of device names.
// These are PCIe root port devices that use the pcieport driver.
func (fs FS) RootPortDevices() ([]string, error) {
	var res []string
	path := fs.sys.Path(pcieportDriverPath)

	devices, err := os.ReadDir(path)
	if err != nil {
		return res, fmt.Errorf("cannot access dir %q: %w", path, err)
	}

	for _, deviceDir := range devices {
		if deviceDir.Type().IsRegular() {
			continue
		}
		res = append(res, deviceDir.Name())
	}

	return res, nil
}

// RootPortAerCounters returns root port AER counters for all root port devices
// read from /sys/bus/pci/drivers/pcieport.
func (fs FS) RootPortAerCounters() (AllRootPortAerCounters, error) {
	devices, err := fs.RootPortDevices()
	if err != nil {
		return nil, err
	}

	allRootPortAerCounters := AllRootPortAerCounters{}
	for _, deviceName := range devices {
		deviceDir := fs.sys.Path(pcieportDriverPath, deviceName)
		counters, err := parseRootPortAerCounters(deviceDir)
		if err != nil {
			return nil, err
		}
		if counters == nil {
			continue
		}
		allRootPortAerCounters[deviceName] = *counters
	}

	return allRootPortAerCounters, nil
}

// parseRootPortAerCounters parses root port AER error counters from
// /sys/bus/pci/drivers/pcieport/<device>/aer_rootport_total_err_* files.
// Returns nil if AER is not supported for the device.
func parseRootPortAerCounters(deviceDir string) (*RootPortAerCounters, error) {
	filenames := []string{
		"aer_rootport_total_err_cor",
		"aer_rootport_total_err_nonfatal",
		"aer_rootport_total_err_fatal",
	}

	if _, err := os.Stat(filepath.Join(deviceDir, "aer_rootport_total_err_cor")); os.IsNotExist(err) {
		return nil, nil
	}

	rootportcounter := RootPortAerCounters{}

	for _, filename := range filenames {
		var fieldValue uint64
		path := filepath.Join(deviceDir, filename)
		value, err := util.SysReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", path, err)
		}
		valueStr := strings.TrimSpace(string(value))
		if valueStr != "" {
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing %s: %w", filename, err)
			}
			fieldValue = v
		}

		switch filename {
		case "aer_rootport_total_err_cor":
			rootportcounter.TotalErrCor = fieldValue
		case "aer_rootport_total_err_nonfatal":
			rootportcounter.TotalErrNonFatal = fieldValue
		case "aer_rootport_total_err_fatal":
			rootportcounter.TotalErrFatal = fieldValue
		}
	}

	return &rootportcounter, nil
}
