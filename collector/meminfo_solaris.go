// Copyright 2017 Tomi Engel
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

// +build solaris

// The node_exporter output for Darwin and Linux has basically nothing in common. They provide similar numbers
// but under very different names.
// Since Linux seems the most common server system our Solaris module will try to map to the Linux identifiers
// TOCHECK: Should we provide precise native identifiers too?
//
// However .. here is an attempt to map the few Darin metrics to their Linux counterparts
//
// Darin                                    Linux
//
// node_memory_active_bytes_total
// node_memory_bytes_total                  node_memory_MemTotal
// node_memory_free_bytes_total
// node_memory_inactive_bytes_total         node_memory_Inactive
// node_memory_swapped_in_pages_total
// node_memory_swapped_out_pages_total
// node_memory_wired_bytes_total
//
//

package collector

import (
	//	"encoding/binary"
	"fmt"
	//	"syscall"
	//	"unsafe"

	"github.com/siebenmann/go-kstat"
	//	"golang.org/x/sys/unix"
	//	"time"
)

func getNamedUint64Val(ks *kstat.KStat, name string) uint64 {
	n, err := ks.GetNamed(name)
	if err != nil {
		// log.Fatalf("getting '%s' from %s: %s", name, ks, err)
	}
	if n.Type != kstat.Uint64 {
		// log.Fatalf("Named value is not od Uint64 type: '%s', %v", name, ks)
	}
	return n.UintVal
}

//
//
func (c *meminfoCollector) getMemInfo() (map[string]float64, error) {

	var (
		memInfo = map[string]float64{}

		// Hardcoded page size .. until we find the proper way to read it
		solarisPageSize uint64 = 4096
	)

	// TODO: Check .. should we cache those tokens for faster access ?
	//
	kstatToken, err := kstat.Open()
	if err != nil {
		return nil, fmt.Errorf("Failed during kstat.Open() with: %s", err)
	}

	kstatTable, err := kstatToken.Lookup("unix", 0, "system_pages")
	if err != nil {
		return nil, err
	}

	// A full in depth documentation of the individual kstat values seems hard to be found.
	// There are some (old) books on the subject
	//
	// - Alexandre Borges: "Oracle Solaris 11 Advanced Administration Cookbook"
	// - AEleen Frisch: "Unix System Administration"
	//
	// As they say .. Read the source Luke! .. the true answers are most likely all within:
	//
	//   https://github.com/joyent/illumos-joyent/blob/master/usr/src/uts/common/os/kstat_fr.c
	//
	// To discover more values you can poke around in a system for example with:
	//
	//   kstat -pc "pages"
	//   kstat -pc "vm"
	//   vmstat -s
	//
	// Here are some general observations about what we think we know so far about "memory" indicators.
	// Values with the "mdb::memstat" prefix are the ones reported by mdb tool.
	//
	// unix:0:system_pages:availrmem .. "amount of unlocked memory available for allocation"
	// unix:0:system_pages:pagestotal - unix:0:system_pages:pageslocked = unix:0:system_pages:availrmem
	// unix:0:system_pages:physmem = unix:0:system_pages:pagestotal + 1
	// unix:0:system_pages:freemem = unix:0:system_pages:pagesfree
	//
	// unix:0:system_pages:pp_kernel = approx. mdb::memstat:Kernel + mdb::memstat:ZFS_File_Data"
	//                                 it is slightly bigger because it is a "counter" and not a "gauge"
	//
	// zfs:0:arcstats:size .. "ZFS ARC cache bytes used within the mdb::memstat:Kernel .. but _not_ including mdb::memstat:ZFS_File_Data"

	// bytes_total
	// - the amount or bytes the system has in total
	//
	memInfo["bytes_total"] = float64(getNamedUint64Val(kstatTable, "physmem") * solarisPageSize)

	// wired_bytes_total
	// - basically what we see in "mdb::memstat" under the label "Kernel".
	//   This also includes the ZFS ARC cache.
	//

	// free_bytes_total
	// - basically what we see in "mdb::memstat" under the label "Free (freelist)".
	//   The "Free (cachelist)" could be freed up too, but most likely only under memory
	//   pressure, which is why we would not count it as "free".
	// - the kstat "unix:0:system_pages:freemem" does not seem to produce identical values
	//   to the memstat but it seems to be in the same ballpark.
	//
	memInfo["free_bytes_total"] = float64(getNamedUint64Val(kstatTable, "freemem") * solarisPageSize)

	// swapped_in_pages_total
	// - on Darwin this is reported in bytes, not pages! "Pageins" * "Page size (4096 bytes)"
	//
	//memInfo["swapped_in_pages_total"] = float64(getNamedUint64Val(kstatTable, "physmem") * solarisPageSize)

	kstatToken.Close()

	return memInfo, nil

	//	This is from the Darwin code:
	//
	// return map[string]float64{
	//		"active_bytes_total":      float64(ps * vmstat.active_count),
	//		"inactive_bytes_total":    float64(ps * vmstat.inactive_count),
	//		"wired_bytes_total":       float64(ps * vmstat.wire_count),
	//		"free_bytes_total":        float64(ps * vmstat.free_count),
	//		"swapped_in_pages_total":  float64(ps * vmstat.pageins)
	//		"swapped_out_pages_total": float64(ps * vmstat.pageouts),
	//		"bytes_total":             float64(total),
	//	}, nil

}
