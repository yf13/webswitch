// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"strconv"
	"github.com/yf13/webswitch"
)

// enum of hub command codes
const (
	CMD_HOST_CNT  = 0
	CMD_PLUG_IN   = 1
	CMD_PLUG_OUT  = 2
	CMD_PLUG_DUMP = 3
)

// base of numeric request id
const REQ_ID_BASE = 10

// hub management command
// TODO: review if "string" is best carrier for replies.
type HubCommand struct {
	cmd      uint8         // the command code as above CMD_* constants
	conn     *PlugConn     // the plug conn to add/drop
	reply_ch chan<- string // the chan to accept command response
}

// client Request contains original client request and a reply-to chan
type ClientRequest struct {
	req      *http.Request        // original plus x-forwared-for
	reply_ch chan<- *PlugResponse // chan to accept response
}

// Hub exchanges messages between clients and plugs
type Hub struct {
	// client requests queue
	req_queue chan *ClientRequest
	// plug responses queue
	rsp_queue chan *PlugResponse
	// command queue for query/register/unregister
	cmd_queue chan *HubCommand
	// registered plugs for vhosts,
	// each host can have bundles of plugs for different limits
	plugs PlugRegistry
	// pending requests and their replying chans
	pending_reqs map[uint64]chan<- *PlugResponse
	// the request id since start of the hub
	req_id uint64
}

// the signleton switch hub
var hub = &Hub{}

// Register a plug to the hub.
// returns the number registries added
func (h *Hub) register(plug *PlugConn) int {
	reply_ch := make(chan string, 1)
	cmd := &HubCommand{CMD_PLUG_IN, plug, reply_ch}
	h.cmd_queue <- cmd
	reply, _ := <-reply_ch
	count, _ := strconv.ParseInt(reply, 10, 32)
	return int(count)
}

// unregister a plug connection from the hub
// returns number of dropped entries
func (h *Hub) unregister(plug *PlugConn) int {
	reply_ch := make(chan string, 1)
	cmd := &HubCommand{CMD_PLUG_OUT, plug, reply_ch}
	h.cmd_queue <- cmd
	reply, _ := <-reply_ch
	count, _ := strconv.ParseInt(reply, 10, 32)
	return int(count)
}

// query max number of registered plugs matching given hosts,
// assuming input hosts list has no duplicates.
// empty input lists will get number of total registered hosts
func (h *Hub) hosts_count(hosts []string) int {
	reply_ch := make(chan string, 1)
	plug := &PlugConn{hosts, nil, nil, 0, 0, 0, 0}
	cmd := &HubCommand{CMD_HOST_CNT, plug, reply_ch}
	h.cmd_queue <- cmd
	reply, _ := <-reply_ch
	count, _ := strconv.ParseInt(reply, 10, 32)
	return int(count)
}

// query registry status
func (h *Hub) status_query(ident bool) string {
	reply_ch := make(chan string, 1)
	cmd := &HubCommand{CMD_PLUG_DUMP, nil, reply_ch}
	h.cmd_queue <- cmd
	status, _ := <-reply_ch
	return status
}

// The switching and management logic of the hub
//
//	- for req, assign reqId, forward to plug conn and keep pending Ids;
// - for rsp, find reply_to chan and forward;
// - for cmd, handles query/register/unregister;
// - log errors and maintain statistics;
//
func (h *Hub) run() {

	// initialize request id
	h.req_id = 0
	h.req_queue = make(chan *ClientRequest, 10)
	h.rsp_queue = make(chan *PlugResponse, 10)
	h.pending_reqs = make(map[uint64]chan<- *PlugResponse)
	h.cmd_queue = make(chan *HubCommand, 1)

	// close all pending requests upon end of this switch routine
	defer func() {
		for _, v := range h.pending_reqs {
			close(v)
		}
	}()

	// error for unknown hosts
	errNotFound := &PlugResponse{
		webswitch.QuickResponse(http.StatusNotFound, nil), nil}
	errReqTooBig := &PlugResponse{
		webswitch.QuickResponse(http.StatusRequestEntityTooLarge, nil), nil}

	// The main switch loop
	for {

		select {
		// ==== incoming client request
		case cr, ok := <-h.req_queue:
			if ok {
				h.req_id += 1
				if _, ok = h.plugs.Hosts[cr.req.Host]; ok {
					if pe := h.plugs.alloc(cr.req); pe != nil {
						log.Println("found plug for", cr.req.Host)
						s := strconv.FormatUint(h.req_id, REQ_ID_BASE)
						cr.req.Header.Add(webswitch.HEADER_REQUEST_ID, s)
						cr.req.Header.Add(webswitch.HEADER_FORWARD_FOR, cr.req.RemoteAddr)
						pe.forward(cr.req)
						log.Printf("fwrd req#%d to plug", h.req_id)
						// keep request id with its reply_ch
						h.pending_reqs[h.req_id] = cr.reply_ch
					} else {
						cr.reply_ch <- errReqTooBig
						close(cr.reply_ch)
						log.Printf("too big req#%d!", h.req_id)
					}
				} else {
					// no plug available, deny immediately
					cr.reply_ch <- errNotFound
					close(cr.reply_ch)
					log.Printf("host not found for req#%d!", h.req_id)
				}
			} else {
				// web client routines normally won't close the chan
				log.Fatal("reqQueue closed unexpectedly!")
				return
			}

		// ==== incoming plug response
		case pr, ok := <-h.rsp_queue:
			if ok {
				// retrieve request id from the response
				rspId, _ := strconv.ParseUint(webswitch.ResponseId(pr.Resp),
					REQ_ID_BASE, 64)
				log.Printf("rcvd plug rsp#%d", rspId)
				if ch, ok := h.pending_reqs[rspId]; ok {
					delete(h.pending_reqs, rspId)
					ch <- pr
					close(ch)
					log.Printf("rply rsp#%d, %d pending", rspId, len(h.pending_reqs))
				} else {
					log.Println("unsolicited rsp: %v", pr.Resp)
					pr.Close()
				}
			} else {
				// rsp queue closed, should not happen by design
				log.Fatal("rspQueue closed unexpectedly!")
				return
			}

		// ==== incoming command requests
		case cr, ok := <-h.cmd_queue:
			if ok {
				switch cr.cmd {
				// query total registered plugs for given hosts
				case CMD_HOST_CNT:
					count := 0
					if 0 == len(cr.conn.hosts) {
						count, _ = h.plugs.size()
					} else {
						// query how many hosts already registered
						for _, host := range cr.conn.hosts {
							if _, ok := h.plugs.Hosts[host]; ok {
								count += 1
							}
						}
					}
					cr.reply_ch <- strconv.FormatInt(int64(count), 10)
				// plug in a conn
				case CMD_PLUG_IN:
					count := h.plugs.register(cr.conn)
					cr.reply_ch <- strconv.FormatInt(int64(count), 10)
				// unplug a conn
				case CMD_PLUG_OUT:
					// close the obuf if the plug to finish its writer loop
					close(cr.conn.obuf)
					// remove the plug from hub's map
					count := h.plugs.unregister(cr.conn)
					cr.reply_ch <- strconv.FormatInt(int64(count), 10)
				// return a dump of the registry status
				case CMD_PLUG_DUMP:
					dump, _ := h.plugs.dump(false)
					cr.reply_ch <- dump
				default:
					log.Printf("Unknown command request %v!", cr)
				}
				// return results to the caller
				close(cr.reply_ch)
			} else {
				// plug req queue closed --- panic
				log.Fatal("command queue closed unexpectedly!")
				return
			}
			// TODO: add default timer case to clean up reqs pending too long
		}
	}
}

// PlugResponse contains the plug response including the HTTP response and
// a chan to end of use. The final consumer of this response shall call the
// Close() method when the response is no longer needed.
type PlugResponse struct {
	Resp  *http.Response // the HTTP response
	_done chan bool      // chan for end of use, use Close()
}

// Close the plug response after use. The response should never be used
// after this.
func (pr *PlugResponse) Close() {
	// close the HTTP response
	if pr.Resp != nil {
		if pr.Resp.Body != nil {
			pr.Resp.Body.Close()
		}
	}
	// indict the plug reader to continue
	if pr._done != nil {
		pr._done <- true
	}
}
