// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*

This is the web switch frontend hub.

The hub accepts secure connections from plugs as well as standard
web (http/https) connections from visitors. It then forwards visitor'
requests to plugged backend web servers and relays responses back.

The hub can have multiple visitor ports (http/https) for visitors and one port
for plugs. These can be specified through command line options.

Upon start, one listener will be started for one visitor/plug port.
Then for each plug connection, one reader and one writer routine will be
created.

The hub maintains a central registry of plugs organized by host names and
message limits. This way, message of different sizes will be passed to
different plugs so that big messages don't block small ones. Multiple plugs
with different limit can exist for one host, and one plug can support
multiple hosts.

The hub should provide web query access for latest status of its central
registry.

Then for each web client request, there is 1 routine created and exist
until the request is done.

The hub needs have proper subjectAlternativeNames fields in its cerficate
so that web clients can make HTTPS connections successfully. This nomrally
means all hub hostnames known to  visitors and plugs should be listed in hub's
certificate.

*/
package main
