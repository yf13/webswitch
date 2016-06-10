// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package webswitch

import (
	"net/http"
)

// constants for header names and subprotocol ids
const (
	HEADER_PROXY_FOR     = "X-Proxy-For"
	HEADER_ORIGIN        = "Origin"
	HEADER_FORWARD_FOR   = "X-Forwarded-For"
	HEADER_REQUEST_ID    = "X-Webx-Request-Id"
	HEADER_MESSAGE_LIMIT = "X-Webx-Message-Limit"
	HEADER_CONTENT_LEN   = "Content-Length"

	SUB_PROTOCOL_WEBX  = "webx"
	MESSAGE_LIMIT_BASE = 10
	HUB_RESOURCE_NAME  = "/_webx"
)

// Hop-by-hop headers in 13.5.1 of RFC2616
var HOP_HEADERS = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// clean hop headers in src
func CleanHopHeaders(src *http.Header) {
	for _, h := range HOP_HEADERS {
		src.Del(h)
	}
}

// Copy http header entries from source to dst.
func CopyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Prepare a quick resp using input status code for the given request
func QuickResponse(code int, req *http.Request) *http.Response {
	rsp := &http.Response{StatusCode: code}
	// copy request id if available
	if id := RequestId(req); id != "" {
		rsp.Header = make(http.Header)
		rsp.Header.Add(HEADER_REQUEST_ID, id)
	}
	return rsp
}

// return the request id string or empty string if undefined
func RequestId(req *http.Request) string {
	id := ""
	if nil != req && nil != req.Header {
		id = req.Header.Get(HEADER_REQUEST_ID)
	}
	return id
}

// return the response id string or empty string if undefined
func ResponseId(rsp *http.Response) string {
	id := ""
	if nil != rsp && nil != rsp.Header {
		id = rsp.Header.Get(HEADER_REQUEST_ID)
	}
	return id
}
