// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
)

func Test_regs(t *testing.T) {
	hosts := []string{"ibm.com", "hp.com", "dell.com", "java.cn"}
	pc1 := &PlugConn{hosts[:2], nil, nil, 5000, 0, 0, 0}
	pc2 := &PlugConn{hosts[1:3], nil, nil, 0, 0, 0, 0}

	reg := PlugRegistry{}

	// try find
	pe := reg.alloc_params(hosts[0], 5001)
	if pe != nil {
		t.Error("expected nil but got", pe)
	}

	// try unregister before register
	n := reg.unregister(pc1)
	if n != 0 {
		t.Error(n, "!=", 0)
	}

	// test register
	n = reg.register(pc1)
	if n != len(pc1.hosts) {
		t.Error(n, "!=", len(pc1.hosts))
	}
	if d, err := reg.dump(true); err == nil {
		t.Log(d)
	} else {
		t.Error("dump error: ", err)
	}
	if hnum, cnum := reg.size(); hnum != 2 || cnum != 1 {
		t.Error("expected 2,1 ", "got", hnum, ",", cnum)
	}

	// regisrter another plug
	n = reg.register(pc2)
	if n != len(pc2.hosts) {
		t.Error(n, "!=", len(pc2.hosts))
	}
	if hnum, cnum := reg.size(); hnum != 3 || cnum != 2 {
		t.Error("expected 3,2 ", "got", hnum, ",", cnum)
	}
	if d, err := reg.dump(true); err == nil {
		t.Log(d)
	} else {
		t.Error("dump error: ", err)
	}

	// try find
	pe = reg.alloc_params(hosts[0], 5001)
	if pe != nil {
		t.Error("expected nil but got", pe)
	}

	// try find
	pe = reg.alloc_params(hosts[1], 2000)
	if pe == nil || pe.Conn != pc1 {
		t.Error("expected", pc1, " Got", pe)
	}

	// try use the entry, since no obuf, it can't be used successfully
	lastUse := pe.Uses
	if ok := pe.forward(nil); ok || pe.Uses != lastUse {
		t.Error("use count should be:", lastUse)
	}

	// try unregister
	n = reg.unregister(pc1)
	if n != len(pc1.hosts) {
		t.Error(n, "!=", len(pc1.hosts))
	}
	if hnum, cnum := reg.size(); hnum != 2 || cnum != 1 {
		t.Error("expected 2,1 ", "got", hnum, ",", cnum)
	}

	// try find again
	pe = reg.alloc_params(hosts[1], 0)
	if pe == nil || pe.Conn != pc2 {
		t.Error("expected", pc2, " Got", pe)
	}

	// try unregister again
	n = reg.unregister(pc1)
	if n > 0 {
		t.Error(n, "!=", 0)
	}

	n = reg.unregister(pc2)
	if n != len(pc2.hosts) {
		t.Error(n, "!=", len(pc2.hosts))
	}
	if hnum, cnum := reg.size(); hnum != 0 || cnum != 0 {
		t.Error("expected 0,0 ", "got", hnum, ",", cnum)
	}

}
