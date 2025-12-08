package sysfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// PciDeviceAerCounters contains generic AER counters from files in /sys/bus/pci/devices/<Location>
type PciDeviceAerCounters struct {
	Correctable              CorrectableAerCounters
	Fatal                    UncorrectableAerCounters
	NonFatal                 UncorrectableAerCounters
	RootPortTotalErrCor      *uint64 // aer_rootport_total_err_cor (nil if file doesn't exist)
	RootPortTotalErrFatal    *uint64 // aer_rootport_total_err_fatal (nil if file doesn't exist)
	RootPortTotalErrNonFatal *uint64 // aer_rootport_total_err_nonfatal (nil if file doesn't exist)
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

// AllAerCounters is collection of AER counters for every interface (iface) in /sys/bus/pci/devices.
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

	// Root port files are optional - parseRootPortAerCounters sets pointers to nil if files don't exist
	err = parseRootPortAerCounters(deviceDir, &counters)
	if err != nil {
		return nil, err
	}

	return &counters, nil
}

// parseAerCounters scans predefined files in /sys/bus/pci/devices/<location> directory and gets their contents.
func (pci *PciDevice) AerCounters(fs FS) (*PciDeviceAerCounters, error) {
	deviceName := fmt.Sprintf("%04x:%02x:%02x.%x", pci.Location.Segment, pci.Location.Bus, pci.Location.Device, pci.Location.Function)
	deviceDir := fs.sys.Path(pciDevicesPath, deviceName)

	return parseAerCounters(deviceDir)
}

// parseRootPortAerCounters parses root port AER error counters from
// /sys/bus/pci/devices/<location>/aer_rootport_total_err_* files.
// If a file doesn't exist, the corresponding pointer field is set to nil.
func parseRootPortAerCounters(deviceDir string, counters *PciDeviceAerCounters) error {

	// Parse aer_rootport_total_err_cor
	path := filepath.Join(deviceDir, "aer_rootport_total_err_cor")
	value, err := util.SysReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, set to nil
			counters.RootPortTotalErrCor = nil
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
			counters.RootPortTotalErrCor = &v
		} else {
			// Empty value, set to nil
			counters.RootPortTotalErrCor = nil
		}
	}

	// Parse aer_rootport_total_err_fatal
	path = filepath.Join(deviceDir, "aer_rootport_total_err_fatal")
	value, err = util.SysReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, set to nil
			counters.RootPortTotalErrFatal = nil
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
			counters.RootPortTotalErrFatal = &v
		} else {
			// Empty value, set to nil
			counters.RootPortTotalErrFatal = nil
		}
	}

	// Parse aer_rootport_total_err_nonfatal
	path = filepath.Join(deviceDir, "aer_rootport_total_err_nonfatal")
	value, err = util.SysReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist, set to nil
			counters.RootPortTotalErrNonFatal = nil
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
			counters.RootPortTotalErrNonFatal = &v
		} else {
			// Empty value, set to nil
			counters.RootPortTotalErrNonFatal = nil
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
