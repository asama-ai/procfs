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

func TestRootPortAerCounters(t *testing.T) {
	fs, err := NewFS(sysTestFixtures)
	if err != nil {
		t.Fatal(err)
	}

	got, err := fs.RootPortAerCounters()
	if err != nil {
		t.Fatalf("failed to get root port AER counters: %v", err)
	}

	want := AllRootPortAerCounters{
		"0000:00:02.1": RootPortAerCounters{
			TotalErrCor:      1,
			TotalErrFatal:    2,
			TotalErrNonFatal: 3,
		},
		"0000:00:03.0": RootPortAerCounters{
			TotalErrCor:      4,
			TotalErrFatal:    5,
			TotalErrNonFatal: 6,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected diff (-want +got):\n%s", diff)
	}
}
