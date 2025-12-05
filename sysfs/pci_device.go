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

// PciPowerState represents the power state of a PCI device.
type PciPowerState string

const (
	PciPowerStateUnknown PciPowerState = "unknown"
	PciPowerStateError   PciPowerState = "error"
	PciPowerStateD0      PciPowerState = "D0"
	PciPowerStateD1      PciPowerState = "D1"
	PciPowerStateD2      PciPowerState = "D2"
	PciPowerStateD3Hot   PciPowerState = "D3hot"
	PciPowerStateD3Cold  PciPowerState = "D3cold"
)

// String returns the string representation of the power state.
func (p PciPowerState) String() string {
	return string(p)
}

const pciDevicesPath = "bus/pci/devices"

// PciDeviceLocation represents the location of the device attached.
// "0000:00:00.0" represents Segment:Bus:Device.Function .
type PciDeviceLocation struct {
	Segment  int
	Bus      int
	Device   int
	Function int
}

func (pdl PciDeviceLocation) String() string {
	return fmt.Sprintf("%04x:%02x:%02x:%x", pdl.Segment, pdl.Bus, pdl.Device, pdl.Function)
}

// DirectoryName returns the location in filesystem directory name format (with dot instead of last colon).
// For example, "0000:01:00.0" instead of "0000:01:00:0".
// func (pdl PciDeviceLocation) DirectoryName() string {
// 	return fmt.Sprintf("%04x:%02x:%02x.%x", pdl.Segment, pdl.Bus, pdl.Device, pdl.Function)
// }

func (pdl PciDeviceLocation) Strings() []string {
	return []string{
		fmt.Sprintf("%04x", pdl.Segment),
		fmt.Sprintf("%02x", pdl.Bus),
		fmt.Sprintf("%02x", pdl.Device),
		fmt.Sprintf("%x", pdl.Function),
	}
}

// PciDevice contains info from files in /sys/bus/pci/devices for a
// single PCI device.
type PciDevice struct {
	Location       PciDeviceLocation
	ParentLocation *PciDeviceLocation

	Class           uint32 // /sys/bus/pci/devices/<Location>/class
	Vendor          uint32 // /sys/bus/pci/devices/<Location>/vendor
	Device          uint32 // /sys/bus/pci/devices/<Location>/device
	SubsystemVendor uint32 // /sys/bus/pci/devices/<Location>/subsystem_vendor
	SubsystemDevice uint32 // /sys/bus/pci/devices/<Location>/subsystem_device
	Revision        uint32 // /sys/bus/pci/devices/<Location>/revision

	NumaNode *int32 // /sys/bus/pci/devices/<Location>/numa_node

	MaxLinkSpeed     *float64 // /sys/bus/pci/devices/<Location>/max_link_speed
	MaxLinkWidth     *float64 // /sys/bus/pci/devices/<Location>/max_link_width
	CurrentLinkSpeed *float64 // /sys/bus/pci/devices/<Location>/current_link_speed
	CurrentLinkWidth *float64 // /sys/bus/pci/devices/<Location>/current_link_width

	SriovDriversAutoprobe *bool   // /sys/bus/pci/devices/<Location>/sriov_drivers_autoprobe
	SriovNumvfs           *uint32 // /sys/bus/pci/devices/<Location>/sriov_numvfs
	SriovOffset           *uint32 // /sys/bus/pci/devices/<Location>/sriov_offset
	SriovStride           *uint32 // /sys/bus/pci/devices/<Location>/sriov_stride
	SriovTotalvfs         *uint32 // /sys/bus/pci/devices/<Location>/sriov_totalvfs
	SriovVfDevice         *uint32 // /sys/bus/pci/devices/<Location>/sriov_vf_device
	SriovVfTotalMsix      *uint64 // /sys/bus/pci/devices/<Location>/sriov_vf_total_msix

	D3coldAllowed *bool          // /sys/bus/pci/devices/<Location>/d3cold_allowed
	PowerState    *PciPowerState // /sys/bus/pci/devices/<Location>/power_state
}

// CorrectableAerCounters contains values from /sys/bus/pci/devices/<Location>/aer_dev_correctable
type CorrectableAerCounters struct {
	RxErr       uint64
	BadTLP      uint64
	BadDLLP     uint64
	Rollover    uint64
	Timeout     uint64
	NonFatalErr uint64
	CorrIntErr  uint64
	HeaderOF    uint64
}

// UncorrectableAerCounters contains values from /sys/bus/pci/devices/<Location>/aer_dev_[non]fatal
// for single interface (iface).
type UncorrectableAerCounters struct {
	Undefined        uint64
	DLP              uint64
	SDES             uint64
	TLP              uint64
	FCP              uint64
	CmpltTO          uint64
	CmpltAbrt        uint64
	UnxCmplt         uint64
	RxOF             uint64
	MalfTLP          uint64
	ECRC             uint64
	UnsupReq         uint64
	ACSViol          uint64
	UncorrIntErr     uint64
	BlockedTLP       uint64
	AtomicOpBlocked  uint64
	TLPBlockedErr    uint64
	PoisonTLPBlocked uint64
}

// PciDeviceAerCounters contains generic AER counters from files in /sys/bus/pci/devices/<Location>
type PciDeviceAerCounters struct {
	Correctable              CorrectableAerCounters
	Fatal                    UncorrectableAerCounters
	NonFatal                 UncorrectableAerCounters
	RootPortTotalErrCor      uint64 // aer_rootport_total_err_cor
	RootPortTotalErrFatal    uint64 // aer_rootport_total_err_fatal
	RootPortTotalErrNonFatal uint64 // aer_rootport_total_err_nonfatal
}

// AllAerCounters is collection of AER counters for every interface (iface) in /sys/bus/pci/devices.
// The map keys are interface (iface) names.
type AllAerCounters map[string]AerCounters

func (pd PciDevice) Name() string {
	return pd.Location.String()
}

// PciDevices is a collection of every PCI device in
// /sys/bus/pci/devices .
//
// The map keys are the location of PCI devices.
type PciDevices map[string]PciDevice

// PciDevices returns info for all PCI devices read from
// /sys/bus/pci/devices .
func (fs FS) PciDevices() (PciDevices, error) {
	path := fs.sys.Path(pciDevicesPath)

	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	pciDevs := make(PciDevices, len(dirs))
	for _, d := range dirs {
		device, err := fs.parsePciDevice(d.Name())
		if err != nil {
			return nil, err
		}

		pciDevs[device.Name()] = *device
	}

	return pciDevs, nil
}

func parsePciDeviceLocation(loc string) (*PciDeviceLocation, error) {
	locs := strings.Split(loc, ":")
	if len(locs) != 3 {
		return nil, fmt.Errorf("invalid location '%s'", loc)
	}
	locs = append(locs[0:2], strings.Split(locs[2], ".")...)
	if len(locs) != 4 {
		return nil, fmt.Errorf("invalid location '%s'", loc)
	}

	seg, err := strconv.ParseInt(locs[0], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid segment: %w", err)
	}
	bus, err := strconv.ParseInt(locs[1], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid bus: %w", err)
	}
	device, err := strconv.ParseInt(locs[2], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid device: %w", err)
	}
	function, err := strconv.ParseInt(locs[3], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid function: %w", err)
	}

	return &PciDeviceLocation{
		Segment:  int(seg),
		Bus:      int(bus),
		Device:   int(device),
		Function: int(function),
	}, nil
}

// Parse one PCI device
// Refer to https://docs.kernel.org/PCI/sysfs-pci.html
func (fs FS) parsePciDevice(name string) (*PciDevice, error) {
	path := fs.sys.Path(pciDevicesPath, name)
	// the file must be symbolic link.
	realPath, err := os.Readlink(path)
	if err != nil {
		return nil, fmt.Errorf("failed to readlink: %w", err)
	}

	// parse device location from realpath
	// like "../../../devices/pci0000:00/0000:00:02.5/0000:04:00.0"
	deviceLocStr := filepath.Base(realPath)
	parentDeviceLocStr := filepath.Base(filepath.Dir(realPath))

	deviceLoc, err := parsePciDeviceLocation(deviceLocStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse device location:%q %w", deviceLoc, err)
	}

	// the parent device may have "pci" prefix.
	// this is not pci device like bridges.
	// we ignore such location to avoid confusion.
	// TODO: is it really ok?
	var parentDeviceLoc *PciDeviceLocation
	if !strings.HasPrefix(parentDeviceLocStr, "pci") {
		parentDeviceLoc, err = parsePciDeviceLocation(parentDeviceLocStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent device location %q: %w", parentDeviceLocStr, err)
		}
	}

	device := &PciDevice{
		Location:       *deviceLoc,
		ParentLocation: parentDeviceLoc,
	}

	// These files must exist in a device directory.
	for _, f := range [...]string{"class", "vendor", "device", "subsystem_vendor", "subsystem_device", "revision"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", name, err)
		}
		value, err := strconv.ParseInt(valueStr, 0, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s %q %s: %w", f, valueStr, device.Location, err)
		}

		switch f {
		case "class":
			device.Class = uint32(value)
		case "vendor":
			device.Vendor = uint32(value)
		case "device":
			device.Device = uint32(value)
		case "subsystem_vendor":
			device.SubsystemVendor = uint32(value)
		case "subsystem_device":
			device.SubsystemDevice = uint32(value)
		case "revision":
			device.Revision = uint32(value)
		default:
			return nil, fmt.Errorf("unknown file %q", f)
		}
	}

	for _, f := range [...]string{"max_link_speed", "max_link_width", "current_link_speed", "current_link_width", "numa_node"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read file %q: %w", name, err)
		}

		// Some devices may be NULL or contain 'Unknown' as a value
		// values defined in drivers/pci/probe.c pci_speed_string
		if valueStr == "" || strings.HasPrefix(valueStr, "Unknown") {
			continue
		}

		switch f {
		case "max_link_speed", "current_link_speed":
			// example "8.0 GT/s PCIe"
			values := strings.SplitAfterN(valueStr, " ", 2)
			if len(values) != 2 {
				return nil, fmt.Errorf("invalid value for %s %q %s", f, valueStr, device.Location)
			}
			if values[1] != "GT/s PCIe" {
				return nil, fmt.Errorf("unknown unit for %s %q %s", f, valueStr, device.Location)
			}
			value, err := strconv.ParseFloat(strings.TrimSpace(values[0]), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s %q %s: %w", f, valueStr, device.Location, err)
			}
			v := float64(value)
			switch f {
			case "max_link_speed":
				device.MaxLinkSpeed = &v
			case "current_link_speed":
				device.CurrentLinkSpeed = &v
			}

		case "max_link_width", "current_link_width":
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s %q %s: %w", f, valueStr, device.Location, err)
			}
			v := float64(value)
			switch f {
			case "max_link_width":
				device.MaxLinkWidth = &v
			case "current_link_width":
				device.CurrentLinkWidth = &v
			}

		case "numa_node":
			value, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s %q %s: %w", f, valueStr, device.Location, err)
			}
			v := int32(value)
			device.NumaNode = &v
		}
	}

	// Parse SR-IOV files (these are optional and may not exist for all devices)
	for _, f := range [...]string{"sriov_drivers_autoprobe", "sriov_numvfs", "sriov_offset", "sriov_stride", "sriov_totalvfs", "sriov_vf_device", "sriov_vf_total_msix"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read SR-IOV file %q %s: %w", name, device.Location, err)
		}

		valueStr = strings.TrimSpace(valueStr)
		if valueStr == "" {
			continue
		}

		switch f {
		case "sriov_drivers_autoprobe":
			// sriov_drivers_autoprobe is a boolean (0 or 1)
			value, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV drivers autoprobe %q %s: %w", valueStr, device.Location, err)
			}
			v := value != 0
			device.SriovDriversAutoprobe = &v

		case "sriov_numvfs":
			value, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV numvfs %q %s: %w", valueStr, device.Location, err)
			}
			v := uint32(value)
			device.SriovNumvfs = &v

		case "sriov_offset":
			value, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV offset %q %s: %w", valueStr, device.Location, err)
			}
			v := uint32(value)
			device.SriovOffset = &v

		case "sriov_stride":
			value, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV stride %q %s: %w", valueStr, device.Location, err)
			}
			v := uint32(value)
			device.SriovStride = &v

		case "sriov_totalvfs":
			value, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV totalvfs %q %s: %w", valueStr, device.Location, err)
			}
			v := uint32(value)
			device.SriovTotalvfs = &v

		case "sriov_vf_device":
			value, err := strconv.ParseUint(valueStr, 16, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV vf device %q %s: %w", valueStr, device.Location, err)
			}
			v := uint32(value)
			device.SriovVfDevice = &v

		case "sriov_vf_total_msix":
			value, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SR-IOV vf total msix %q %s: %w", valueStr, device.Location, err)
			}
			v := uint64(value)
			device.SriovVfTotalMsix = &v
		}
	}

	// Parse power management files (these are optional and may not exist for all devices)
	for _, f := range [...]string{"d3cold_allowed", "power_state"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read power management file %q %s: %w", name, device.Location, err)
		}

		valueStr = strings.TrimSpace(valueStr)
		if valueStr == "" {
			continue
		}

		switch f {
		case "d3cold_allowed":
			// d3cold_allowed is a boolean (0 or 1)
			value, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse d3cold_allowed boolean %q %s: %w", valueStr, device.Location, err)
			}
			v := value != 0
			device.D3coldAllowed = &v

		case "power_state":
			// power_state is a string (one of: "unknown", "error", "D0", "D1", "D2", "D3hot", "D3cold")
			powerState := PciPowerState(valueStr)
			device.PowerState = &powerState
		}
	}

	return device, nil
}

// parseAerCounters scans predefined files in /sys/bus/pci/devices/<location> directory and gets their contents.
func parseAerCounters(deviceDir string) (*PciDeviceAerCounters, error) {
	counters := PciDeviceAerCounters{}
	err := parseCorrectableAerCounters(deviceDir, &counters.Correctable)
	if err != nil {
		return nil, err
	}
	err = parseUncorrectableAerCounters(deviceDir, "fatal", &counters.Fatal)
	if err != nil {
		return nil, err
	}
	err = parseUncorrectableAerCounters(deviceDir, "nonfatal", &counters.NonFatal)
	if err != nil {
		return nil, err
	}

	err = parseRootPortAerCounters(deviceDir, &counters)
	if err != nil {
		return nil, err
	}

	return &counters, nil
}

func (pci *PciDevice) AerCounters(fs FS) (*PciDeviceAerCounters, error) {
	deviceName := fmt.Sprintf("%04x:%02x:%02x.%x", pci.Location.Segment, pci.Location.Bus, pci.Location.Device, pci.Location.Function)
	deviceDir := fs.sys.Path(pciDevicesPath, deviceName)

	return parseAerCounters(deviceDir)
}

// parseRootPortAerCounters parses root port AER error counters from
// /sys/bus/pci/devices/<location>/aer_rootport_total_err_* files.
func parseRootPortAerCounters(deviceDir string, counters *PciDeviceAerCounters) error {

	// Parse aer_rootport_total_err_cor
	path := filepath.Join(deviceDir, "aer_rootport_total_err_cor")
	value, err := util.SysReadFile(path)
	if err != nil {
		if canIgnoreError(err) {
		} else {
			return fmt.Errorf("failed to read file %q: %w", path, err)
		}
	} else {
		valueStr := strings.TrimSpace(string(value))
		if valueStr != "" {
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing aer_rootport_total_err_cor: %w", err)
			}
			counters.RootPortTotalErrCor = v
		}
	}

	// Parse aer_rootport_total_err_fatal
	path = filepath.Join(deviceDir, "aer_rootport_total_err_fatal")
	value, err = util.SysReadFile(path)
	if err != nil {
		if canIgnoreError(err) {
		} else {
			return fmt.Errorf("failed to read file %q: %w", path, err)
		}
	} else {
		valueStr := strings.TrimSpace(string(value))
		if valueStr != "" {
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing aer_rootport_total_err_fatal: %w", err)
			}
			counters.RootPortTotalErrFatal = v
		}
	}

	// Parse aer_rootport_total_err_nonfatal
	path = filepath.Join(deviceDir, "aer_rootport_total_err_nonfatal")
	value, err = util.SysReadFile(path)
	if err != nil {
		if canIgnoreError(err) {
		} else {
			return fmt.Errorf("failed to read file %q: %w", path, err)
		}
	} else {
		valueStr := strings.TrimSpace(string(value))
		if valueStr != "" {
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing aer_rootport_total_err_nonfatal: %w", err)
			}
			counters.RootPortTotalErrNonFatal = v
		}
	}

	return nil
}

// parseCorrectableAerCounters parses correctable error counters in
// /sys/bus/pci/devices/<location>/aer_dev_correctable.
func parseCorrectableAerCounters(deviceDir string, counters *CorrectableAerCounters) error {
	path := filepath.Join(deviceDir, "aer_dev_correctable")
	value, err := util.SysReadFile(path)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}

	for line := range strings.SplitSeq(string(value), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("unexpected number of fields: %v", fields)
		}
		counterName := fields[0]
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing value for %s: %w", counterName, err)
		}

		switch counterName {
		case "RxErr":
			counters.RxErr = value
		case "BadTLP":
			counters.BadTLP = value
		case "BadDLLP":
			counters.BadDLLP = value
		case "Rollover":
			counters.Rollover = value
		case "Timeout":
			counters.Timeout = value
		case "NonFatalErr":
			counters.NonFatalErr = value
		case "CorrIntErr":
			counters.CorrIntErr = value
		case "HeaderOF":
			counters.HeaderOF = value
		default:
			continue
		}
	}

	return nil
}

// parseUncorrectableAerCounters parses uncorrectable error counters in
// /sys/bus/pci/devices/<location>/aer_dev_[non]fatal.
func parseUncorrectableAerCounters(deviceDir string, counterType string,
	counters *UncorrectableAerCounters) error {
	path := filepath.Join(deviceDir, "aer_dev_"+counterType)
	value, err := util.ReadFileNoStat(path)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}

	for line := range strings.SplitSeq(string(value), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("unexpected number of fields: %v", fields)
		}
		counterName := fields[0]
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing value for %s: %w", counterName, err)
		}

		switch counterName {
		case "Undefined":
			counters.Undefined = value
		case "DLP":
			counters.DLP = value
		case "SDES":
			counters.SDES = value
		case "TLP":
			counters.TLP = value
		case "FCP":
			counters.FCP = value
		case "CmpltTO":
			counters.CmpltTO = value
		case "CmpltAbrt":
			counters.CmpltAbrt = value
		case "UnxCmplt":
			counters.UnxCmplt = value
		case "RxOF":
			counters.RxOF = value
		case "MalfTLP":
			counters.MalfTLP = value
		case "ECRC":
			counters.ECRC = value
		case "UnsupReq":
			counters.UnsupReq = value
		case "ACSViol":
			counters.ACSViol = value
		case "UncorrIntErr":
			counters.UncorrIntErr = value
		case "BlockedTLP":
			counters.BlockedTLP = value
		case "AtomicOpBlocked":
			counters.AtomicOpBlocked = value
		case "TLPBlockedErr":
			counters.TLPBlockedErr = value
		case "PoisonTLPBlocked":
			counters.PoisonTLPBlocked = value
		default:
			continue
		}
	}

	return nil
}
