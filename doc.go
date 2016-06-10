// Copyright 2015 The WebSwith authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*

Package webx contains common definitions and functions for web switch.

The switch contains frontend switch hub and backend proxy plugs.
The system allows multiple backends to plug into one hub at runtime.

With web switch system, only the hub needs be reachable from web clients,
the backend plugs and real web servers can live deeply in intranet as long
as they can dial to the hub.

Former version only allows one plug for one host, thus larger messages
will block smaller ones. This version allows multiple plugs for one host,
thus messages can be routed to different plugs based on sizes.

Note that each plug still connects to one hub only for simplicity, in cases
where multiple conns are needed to the same hub, a separate plug process
can be started.

*/
package webswitch
