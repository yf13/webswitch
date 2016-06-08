// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"github.com/yf13/webswitch"
)

// a plug entry with a use count. Note this count is different from the one
// inside the plug conn. The latter is for the conn, not for this entry.
// Multiple entries may share the same plug conn since one conn may be
// serving multiple hosts.
type PlugEntry struct {
	Uses uint64    // use count of this entry
	Conn *PlugConn // underlying conn, shared among multiple vhosts
}

// forward HTTP client req to the plug and update the use counters
func (pe *PlugEntry) forward(req *http.Request) bool {
	success := false
	if pe.Conn != nil && pe.Conn.obuf != nil && req != nil {
		pe.Conn.obuf <- req
		pe.Conn.uses += 1
		pe.Uses += 1
		success = true
	}
	return success
}

// The bundle of plugs with the same limit for the same vhost,
// The plugs in the bundle should be used equally, A simple round robin
// usage managed by a "next" index can be a starting point.
//
type PlugBundle struct {
	Limit int64       // the limit, should be same as those of contained plugs
	next  int         // the plug to use next time
	Plugs []PlugEntry // plugs list
}

// map of registered vhosts and plugs,
// Each host has multiple lists sorted by limits, each list can have
// multiple conns with same limit. Thus each conn may appear for multi-hosts,
// so when unregistering, they all need be cleaned.
type PlugRegistry struct {
	Hosts     map[string][]PlugBundle // vhost -> bundle -> entry
	num_plugs int                     // number of conns in registry
	plug_id   int                     // seed for numbering conns
}

// returns the number of hosts and conns
func (reg *PlugRegistry) size() (nhosts int, nconns int) {
	if reg.Hosts != nil {
		return len(reg.Hosts), reg.num_plugs
	}
	return 0, reg.num_plugs
}

// Register a plug for all its vhosts
// returns the number of entries added
func (reg *PlugRegistry) register(plug *PlugConn) int {
	added := 0
	// only accept non-empty vhosts and null id
	if plug != nil && len(plug.hosts) > 0 && plug.Id == 0 {
		// adjust the unlimited plug into max int64
		if plug.limit <= 0 {
			plug.limit = math.MaxInt64
		}
		pe := PlugEntry{0, plug}
		for _, v := range plug.hosts {
			// check if the host is new
			if reg.Hosts == nil {
				reg.Hosts = make(map[string][]PlugBundle)
			}
			if pbl, ok := reg.Hosts[v]; ok {
				// the host is registered, check if a bundle same limit exists,
				// assuming bundles are sorted by limit.
				// TODO use binary search later.
				pos := len(pbl)
				for i, pb := range pbl {
					if pb.Limit >= plug.limit {
						pos = i
						break
					}
				}
				// add the new plug to proper position
				if pos < len(pbl) {
					// check if new bundles should be created or not
					if plug.limit == pbl[pos].Limit {
						// no need for new bundle, just append
						pbl[pos].Plugs = append(pbl[pos].Plugs, pe)
					} else {
						// new bundle should be inserted before pos
						pb := PlugBundle{plug.limit, 0, []PlugEntry{pe}}
						reg.Hosts[v] = make([]PlugBundle, len(pbl)+1)
						copy(reg.Hosts[v], pbl[:pos])
						reg.Hosts[v][pos] = pb
						copy(reg.Hosts[v][pos+1:], pbl[pos:])
					}
				} else {
					// need new bundle for new plug and append to the tail
					pb := PlugBundle{plug.limit, 0, []PlugEntry{pe}}
					reg.Hosts[v] = append(reg.Hosts[v], pb)
				}
			} else {
				// the host never registered
				pb := PlugBundle{plug.limit, 0, []PlugEntry{pe}}
				reg.Hosts[v] = []PlugBundle{pb}
			}
			added += 1
		}
		// assign conn id for the plug to mark it is registered
		reg.plug_id += 1
		plug.Id = reg.plug_id
		reg.num_plugs += 1

		log.Println("plugged in: ", plug)
	} else {
		log.Println("bad plug denied: ", plug)
	}
	return added
}

// dump the registry status as JSON string
// ident controls whether to ident the result
func (reg *PlugRegistry) dump(ident bool) (string, error) {
	var d []byte
	var err error
	if ident {
		d, err = json.MarshalIndent(reg, "", "  ")
	} else {
		d, err = json.Marshal(reg)
	}
	return string(d), err
}

// unregister a plug connection from the hub
// returns number of dropped entries
func (reg *PlugRegistry) unregister(plug *PlugConn) int {
	dropped := 0
	if plug != nil && len(plug.hosts) > 0 && plug.Id > 0 {
		for _, h := range plug.hosts {
			// drop each entry related the plug
			if pbl, ok := reg.Hosts[h]; ok && len(pbl) > 0 {
				// find the right bundle containing the plug
				// TODO: use binary search later
				ndx := len(pbl)
				for i, pb := range pbl {
					if pb.Limit == plug.limit {
						ndx = i
						break
					}
				}
				if ndx < len(pbl) && len(pbl[ndx].Plugs) > 0 {
					i := len(pbl[ndx].Plugs)
					for j, v := range pbl[ndx].Plugs {
						if v.Conn == plug {
							i = j
							break
						}
					}
					if i < len(pbl[ndx].Plugs) {
						// drop the plug entry, together with use counters
						copy(pbl[ndx].Plugs[i:], pbl[ndx].Plugs[i+1:])
						pbl[ndx].Plugs = pbl[ndx].Plugs[:len(pbl[ndx].Plugs)-1]
						dropped += 1
					}
					// check if the bundle is empty
					// TODO: what if backend redial later?
					if len(pbl[ndx].Plugs) < 1 {
						// drop empty bundle
						copy(pbl[ndx:], pbl[ndx+1:])
						pbl = pbl[:len(pbl)-1]
					}
					if len(pbl) < 1 {
						// drop the map entry if no bundles exist
						delete(reg.Hosts, h)
					} else {
						// update the map with reduced bundle list
						reg.Hosts[h] = pbl
					}
				}
			}
		}
		// decrease conn count
		if dropped > 0 {
			reg.num_plugs -= 1
			log.Println("unplugged: ", plug)
		} else {
			log.Println("not found: ", plug)
		}
	}
	return dropped
}

// find a plug proper to handle given message size
// when no proper entry exist, nil will be returned
func (reg *PlugRegistry) alloc_params(host string, size int64) *PlugEntry {
	var plug *PlugEntry = nil
	if pbl, ok := reg.Hosts[host]; ok && len(pbl) > 0 {
		for _, pb := range pbl {
			if pb.Limit >= size && len(pb.Plugs) > 0 {
				plug = &pb.Plugs[pb.next%len(pb.Plugs)]
				pb.next = (pb.next + 1) % len(pb.Plugs)
				break
			}
		}
	}
	return plug
}

// allocate a plug proper to forward the given HTTP request
// when no proper entry exist, nil will be returned
func (reg *PlugRegistry) alloc(req *http.Request) *PlugEntry {
	var plug *PlugEntry = nil
	if req != nil {
		size, _ := strconv.ParseInt(
			req.Header.Get(webswitch.HEADER_CONTENT_LEN), 10, 64)
		plug = reg.alloc_params(req.Host, size)
	}
	return plug
}

// query number of registered hosts
func (reg *PlugRegistry) registered(hosts []string) int {
	return 0
}
