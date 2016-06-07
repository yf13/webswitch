// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"webswitch"
)

// ProxyConn represents the websocket w/ a backend plug proxy
type PlugConn struct {
	// the virtual hosts of this connection
	hosts []string
	// the outgoing request queue to
	obuf chan *http.Request
	// the underlying websocket
	ws *websocket.Conn
	// the limit of message (mainly for client request)
	limit int64
	// the unique id of the conn assigned by the hub
	Id int
	// the time of birth
	birth int64
	// uses count
	uses uint64
}

// outgoing request queue length
const OUT_BUFFER_LENGTH = 5

// connection.Reader reads incoming websocket text messages
// and forward them to the hub
func (c *PlugConn) Reader(h *Hub) {
	// unregister this connection from hub upon errors
	defer func() {
		n := h.unregister(c)
		log.Println(n, "hosts unregistered, total", h.hosts_count(nil))
	}()

	closeCh := make(chan bool, 1)
	pres := &PlugResponse{nil, closeCh}

	for {
		// read message from web socket
		mt, r, err := c.ws.NextReader()
		if err != nil {
			log.Printf("error read plug: %v\n", err)
			break
		}
		if mt != websocket.BinaryMessage {
			// TODO: other types also possible here?
			log.Printf("error plug resp type: %v", mt)
		}
		rsp, err := http.ReadResponse(bufio.NewReader(r), nil)
		if err == nil {
			// forward message to client handler, client should close
			// the rsp.Body finally.
			log.Printf("rcvd plug rsp#%s clen=%d", webswitch.ResponseId(rsp),
				rsp.ContentLength)
			pres.Resp = rsp
			h.rsp_queue <- pres
			// wait until the rsp has been closed
			_ = <-closeCh

		} else {
			log.Printf("error read plug resp: %v", err)
			break
		}
	}
}

// connection.Writer forward incoming request from hub to websocket peer
func (c *PlugConn) Writer() {
	// close the underlying websocket upon done
	defer func() {
		c.ws.Close()
		log.Printf("plug conn %v closed\n", c.ws.RemoteAddr())
	}()

	for {
		if req, ok := <-c.obuf; ok {
			// forward the request to websocket
			w, err := c.ws.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Println("error open plug writer: %v", err)
				return
			}
			// Request.Write/WriteProxy will close the req.Body
			if err := req.WriteProxy(w); err != nil {
				log.Println("error write plug: %v", err)
				return
			}
			w.Close()
			log.Printf("sent req#%s to plug\n", webswitch.RequestId(req))
		} else {
			// obuf has been closed upon hub.unregister()
			c.ws.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}

// customized websocket upgrader that checks subprotocol but not origin
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{webswitch.SUB_PROTOCOL_WEBX},
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Websocket request handler.
//
// For websocket dial request, it checks the parameters and upgrades to
// websocket and register with the switch hub.
//
// For normal request, it returns the internal switch status
//
func handlePlug(w http.ResponseWriter, r *http.Request) {

	// log exit and latest number of proxies...
	defer log.Println("- handlePlug")

	log.Println("+ handlePlug")

	// check request method
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		log.Println("method unsuppported")
		return
	}

	// check if it is a plug request
	if hosts := r.Header[webswitch.HEADER_PROXY_FOR]; len(hosts) > 0 {
		// Update registry accordingly
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("error upgrade:", err)
			// need respond to backend
			http.Error(w, "Upgrade error", 405)
			return
		}
		// the connection has been accepted, register to the hub
		var l int64 = 0
		if n, e := strconv.ParseInt(r.Header.Get(webswitch.HEADER_MESSAGE_LIMIT),
			webswitch.MESSAGE_LIMIT_BASE, 64); e == nil {
			l = n
		}
		c := &PlugConn{
			hosts,
			make(chan *http.Request, OUT_BUFFER_LENGTH),
			ws,
			l, 0, 0, 0,
		}
		n := hub.register(c)
		// start writer loop
		go c.Writer()
		// run reader loop w/ the singleton hub
		go c.Reader(hub)
		log.Println(n, "hosts registered, total is", hub.hosts_count(nil))
	} else {
		// TODO: implement status dump based on switch hub status later
		fmt.Fprintln(w, hub.status_query(false))
	}
}
