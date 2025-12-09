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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPciAerCounters(t *testing.T) {
	fs, err := NewFS(sysTestFixtures)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := fs.PciDevices()
	if err != nil {
		t.Fatal(err)
	}

	device1, ok := devices["0000:01:00:0"]
	if !ok {
		t.Fatal("device 0000:01:00:0 not found")
	}
	got1, err := device1.AerCounters(fs)
	if err != nil {
		t.Fatalf("failed to get AER counters for 0000:01:00:0: %v", err)
	}
	if got1 == nil {
		t.Fatal("AER counters should not be nil for device 0000:01:00:0")
	}

	want1 := &PciDeviceAerCounters{
		Correctable: CorrectableAerCounters{
			RxErr:       1,
			BadTLP:      2,
			BadDLLP:     3,
			Rollover:    4,
			Timeout:     5,
			NonFatalErr: 6,
			CorrIntErr:  7,
			HeaderOF:    8,
		},
		Fatal: UncorrectableAerCounters{
			Undefined:        9,
			DLP:              10,
			SDES:             11,
			TLP:              12,
			FCP:              13,
			CmpltTO:          14,
			CmpltAbrt:        15,
			UnxCmplt:         16,
			RxOF:             17,
			MalfTLP:          18,
			ECRC:             19,
			UnsupReq:         20,
			ACSViol:          21,
			UncorrIntErr:     22,
			BlockedTLP:       23,
			AtomicOpBlocked:  24,
			TLPBlockedErr:    25,
			PoisonTLPBlocked: 26,
		},
		NonFatal: UncorrectableAerCounters{
			Undefined:        27,
			DLP:              28,
			SDES:             29,
			TLP:              30,
			FCP:              31,
			CmpltTO:          32,
			CmpltAbrt:        33,
			UnxCmplt:         34,
			RxOF:             35,
			MalfTLP:          36,
			ECRC:             37,
			UnsupReq:         38,
			ACSViol:          39,
			UncorrIntErr:     40,
			BlockedTLP:       41,
			AtomicOpBlocked:  42,
			TLPBlockedErr:    43,
			PoisonTLPBlocked: 44,
		},
	}

	if diff := cmp.Diff(want1, got1); diff != "" {
		t.Fatalf("unexpected AER counters for device 0000:01:00:0 (-want +got):\n%s", diff)
	}

	device2, ok := devices["0000:a2:00:0"]
	if !ok {
		t.Fatal("device 0000:a2:00:0 not found")
	}
	got2, err := device2.AerCounters(fs)
	if err != nil {
		t.Fatalf("failed to get AER counters for 0000:a2:00:0: %v", err)
	}
	if got2 == nil {
		t.Fatal("AER counters should not be nil for device 0000:a2:00:0")
	}

	want2 := &PciDeviceAerCounters{
		Correctable: CorrectableAerCounters{
			RxErr:       1,
			BadTLP:      2,
			BadDLLP:     3,
			Rollover:    4,
			Timeout:     5,
			NonFatalErr: 6,
			CorrIntErr:  7,
			HeaderOF:    8,
		},
		Fatal: UncorrectableAerCounters{
			Undefined:        9,
			DLP:              10,
			SDES:             11,
			TLP:              12,
			FCP:              13,
			CmpltTO:          14,
			CmpltAbrt:        15,
			UnxCmplt:         16,
			RxOF:             17,
			MalfTLP:          18,
			ECRC:             19,
			UnsupReq:         20,
			ACSViol:          21,
			UncorrIntErr:     22,
			BlockedTLP:       23,
			AtomicOpBlocked:  24,
			TLPBlockedErr:    25,
			PoisonTLPBlocked: 26,
		},
		NonFatal: UncorrectableAerCounters{
			Undefined:        27,
			DLP:              28,
			SDES:             29,
			TLP:              30,
			FCP:              31,
			CmpltTO:          32,
			CmpltAbrt:        33,
			UnxCmplt:         34,
			RxOF:             35,
			MalfTLP:          36,
			ECRC:             37,
			UnsupReq:         38,
			ACSViol:          39,
			UncorrIntErr:     40,
			BlockedTLP:       41,
			AtomicOpBlocked:  42,
			TLPBlockedErr:    43,
			PoisonTLPBlocked: 44,
		},
	}

	if diff := cmp.Diff(want2, got2); diff != "" {
		t.Fatalf("unexpected AER counters for device 0000:a2:00:0 (-want +got):\n%s", diff)
	}
}

func TestPciAerRootPortCounters(t *testing.T) {
	fs, err := NewFS(sysTestFixtures)
	if err != nil {
		t.Fatal(err)
	}

	devices, err := fs.PciDevices()
	if err != nil {
		t.Fatal(err)
	}

	device, ok := devices["0000:00:02:1"]
	if !ok {
		t.Fatal("device 0000:00:02:1 not found")
	}
	got, err := device.AerCounters(fs)
	if err != nil {
		t.Fatalf("failed to get AER counters for 0000:00:02:1: %v", err)
	}
	if got == nil {
		t.Fatal("AER counters should not be nil for device 0000:00:02:1")
	}

	rootPortTotalErrCor := uint64(1)
	rootPortTotalErrFatal := uint64(2)
	rootPortTotalErrNonFatal := uint64(3)

	if *got.RootPortTotalErrCor != rootPortTotalErrCor {
		t.Errorf("RootPortTotalErrCor: want %d, got %d", rootPortTotalErrCor, got.RootPortTotalErrCor)
	}
	if *got.RootPortTotalErrFatal != rootPortTotalErrFatal {
		t.Errorf("RootPortTotalErrFatal: want %d, got %d", rootPortTotalErrFatal, got.RootPortTotalErrFatal)
	}
	if *got.RootPortTotalErrNonFatal != rootPortTotalErrNonFatal {
		t.Errorf("RootPortTotalErrNonFatal: want %d, got %d", rootPortTotalErrNonFatal, got.RootPortTotalErrNonFatal)
	}
}
