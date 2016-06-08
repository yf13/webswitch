// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"io"
	"log"
	"net/http"
	"github.com/yf13/webswitch"
)

// Client http request handler routine
// it finds the proxy registry by host and forwards the request
// with a replyTo chan. Then it waits at replyTo chan for response
// and send it back to the original client.
func handleClient(w http.ResponseWriter, r *http.Request) {
	defer log.Println("- handleClient")
	log.Println("+ handleClient")
	ch := make(chan *PlugResponse)
	// forward request to proxy
	hub.req_queue <- &ClientRequest{r, ch}
	log.Println("sent req to hub")
	r = nil // forget client request
	// wait for response from switch
	if pr, ok := <-ch; ok {
		log.Println("rcvd hub response")
		// clear rsp headers before answering client
		webswitch.CleanHopHeaders(&(pr.Resp.Header))
		webswitch.CopyHeader(w.Header(), pr.Resp.Header)
		w.WriteHeader(pr.Resp.StatusCode)
		if nil != pr.Resp.Body {
			// copy body
			n, err := io.Copy(w, pr.Resp.Body)
			log.Printf("sent %d to clnt, err=%v", n, err)
		}
		pr.Close()
		// ignore trailers for now
	}
}
