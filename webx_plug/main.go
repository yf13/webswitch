// Copyright 2015 The Web Switch Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"github.com/yf13/webswitch"
)

// const
const (
	HUB_REQ_QUEUE_LEN = 1
	HUB_RSP_QUEUE_LEN = 5
)

// command line options
var (
	fe_url     = flag.String("hub", "", "hub's URL to plug into.")
	limit      = flag.Int64("limit", 0, "size limit, 0 is unlimited.")
	key_file   = flag.String("key", "", "plug private key.pem.")
	cert_file  = flag.String("cert", "", "plug public signed cert.crt.")
	ca_file    = flag.String("ca", "", "root CA pem: ca.crt")
	retry_wait = flag.Int("retry", 60, "redial waiting seconds")
	vhosts     = flag.String("hosts", "", "virtual hosts splitted by ','")
	rhosts     = flag.String("rhosts", "", "comma separated real hosts matching vhosts")
	//feLinks=flag.String("links", "0:1", "list of link limit(KB):number, ...")
)

// dial frontend switch hub with specified limit
func dial_hub(feUrl string, limit int64) (*websocket.Conn, error) {
	dialer := websocket.Dialer{} // take default options
	caPool := x509.NewCertPool()
	// load root ca if it is specified
	if pem, err := ioutil.ReadFile(*ca_file); err == nil {
		caPool.AppendCertsFromPEM(pem)
		log.Println("root CAs: ", caPool)
		dialer.TLSClientConfig = &tls.Config{RootCAs: caPool}
	} else {
		log.Println("Error root ca: ", err)
	}
	if "" != *cert_file && "" != *key_file {
		if crt, err := tls.LoadX509KeyPair(*cert_file, *key_file); err == nil {
			if dialer.TLSClientConfig != nil {
				dialer.TLSClientConfig.Certificates = []tls.Certificate{crt}
			} else {
				dialer.TLSClientConfig = &tls.Config{
					Certificates: []tls.Certificate{crt}}
			}
		} else {
			// simply log error and continue
			log.Println("Error key pair: ", err)
		}
	}
	dialer.Subprotocols = []string{webswitch.SUB_PROTOCOL_WEBX}
	h := make(http.Header)
	for _, v := range strings.Split(*vhosts, ",") {
		h.Add(webswitch.HEADER_PROXY_FOR, v)
	}
	if limit > 0 {
		h.Add(webswitch.HEADER_MESSAGE_LIMIT,
			strconv.FormatInt(limit, webswitch.MESSAGE_LIMIT_BASE))
	}
	log.Printf("dialing %s with limit=%d...", feUrl, limit)
	c, _, err := dialer.Dial(feUrl, h)
	if err != nil {
		log.Println("error dial:", err)
	} else {
		log.Println("connected")
	}
	return c, err
}

type HubRequest struct {
	req  *http.Request
	done chan bool
}

// hub request reader.
// It reads incoming request from frontend hub and pass them
// to the main loop via the ch. The ch should be closed when
// reader ends
func hubReader(conn *websocket.Conn, ch chan<- *HubRequest) {
	// always ch before return
	defer close(ch)

	// reader loop
	for {
		mt, r, err := conn.NextReader()
		if err != nil {
			log.Println("error open hub reader:", err)
			conn.Close()
			return
		}
		if mt == websocket.BinaryMessage {
			req, err := http.ReadRequest(bufio.NewReader(r))
			if err != nil {
				// log and ignore this error
				log.Println("warn read hub req:", err)
				continue
			}
			// forward the request to web client
			done := make(chan bool)
			ch <- &HubRequest{req, done}
			// make sure client closed the req fully before
			// reading next one.
			<-done
		} else {
			log.Println("wrong hub req type:", mt)
			continue
		}
	}
}

// hub writer routine.
// It forwards incoming http response to frontend hub.
// it closes the underlying conn before exit
func hubWriter(conn *websocket.Conn, ch <-chan *http.Response) {

	// always close underlying conn
	defer conn.Close()

	// the writer loop
	for {
		// wait incoming response
		rsp, ok := <-ch
		if !ok {
			// response chan closed
			log.Println("exit hub writer")
			break
		}
		if nil != rsp {
			// forward the response to frontend hub
			w, err := conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				// just log and wait for main loop to close me
				log.Println("error open hub writer:", err)
				// the hub should have logic to close pending reqs
			} else {
				rspId := webswitch.ResponseId(rsp)
				// write out response, including body
				if err = rsp.Write(w); err != nil {
					log.Println("error write hub", err)
				}
				// close the writer to output all data
				if err = w.Close(); err != nil {
					log.Println("error close hub writer", err)
				} else {
					log.Printf("sent rsp#%s to hub", rspId)

				}
				// close the resp body anyway
				if rsp.Body != nil {
					rsp.Body.Close()
				}
			}
		}
	}
}

// The web client routine. It executes request against proxied
// web server and prepares response. timeouts are handled as well.
// The channel doesn't need be closed here since it is used by many
// web clients.
func webClient(
	id int,
	req *HubRequest,
	srv string,
	rsp_ch chan<- *http.Response,
	clt_done chan<- int,
) {
	// tell main loop I am done
	defer func() { clt_done <- id }()

	if req != nil && rsp_ch != nil {
		// check the request id for response use
		reqId := req.req.Header.Get(webswitch.HEADER_REQUEST_ID)
		if reqId != "" {
			log.Printf("rcvd %s req#%s\n", req.req.Method, reqId)

			// TODO: avoid parsing everytime
			srvUrl, _ := url.Parse(srv)
			if "" != srvUrl.Scheme {
				req.req.URL.Scheme = srvUrl.Scheme
			}
			if "" != srvUrl.Host {
				req.req.URL.Host = srvUrl.Host
			} else {
				req.req.URL.Host = srv
			}

			// need clear RequestURI in client requests.
			req.req.RequestURI = ""

			rsp, err := http.DefaultClient.Do(req.req)
			req.done <- true
			if err != nil {
				log.Printf("error do req#%s: %v", webswitch.RequestId(req.req), err)
				rsp_ch <- webswitch.QuickResponse(http.StatusInternalServerError,
					req.req)
				log.Printf("sent error rsp#%s\n", reqId)
			} else {
				rsp.Header.Add(webswitch.HEADER_REQUEST_ID, reqId)
				rsp_ch <- rsp
				log.Printf("sent rsp#%s\n", reqId)
			}
		} else {
			rsp_ch <- webswitch.QuickResponse(http.StatusMethodNotAllowed, req.req)
			log.Println("denied req w/o id")
		}
	}
}

/*
// analysis feLinks option and save results in links map
// no longer needed since each plug only creates one link with the hub
// multiple plugs can be used when multiple links are needed
func parseLinks(opt string) map[int]int {
	elements := []string{opt}
	if strings.Contains(opt, ",") {
		elements = strings.Split(opt, ",")
	}
	links := make(map[int]int)
	for _, e := range elements {
		nums := strings.Split(e, ":")
		if len(nums) > 1 {
			if l, er := strconv.Atoi(nums[0]); er == nil {
				if n, er := strconv.Atoi(nums[1]); er == nil {
					if (n > 0) && (l >= 0) {
						if _, ok := links[l]; ok {
							links[l] += n
						} else {
							links[l] = n
						}
					}
				}
			}
		}
	}
	return links
}
*/

// main entrance
func main() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)

	flag.Parse()

	log.Println("version:", APP_VERSION)

	// links := parseLinks(*feLinks)
	// log.Println(*feLinks, "==>", links)

	if nil == fe_url || *fe_url == "" {
		log.Println("missing hub addr")
		return
	}

	if _, err := url.Parse(*fe_url); err != nil {
		log.Println("invalid hub addr", *fe_url)
		return
	}

	// analysis virtual and real hosts options
	vlist, rlist := strings.Split(*vhosts, ","), strings.Split(*rhosts, ",")
	vnum, rnum := len(vlist), len(rlist)
	if (vnum != rnum) || (vnum == 0) {
		log.Println("invalid vhosts/rhosts")
		return
	}
	// preparing hosts map for proxy use purposes
	hosts := make(map[string]string)
	for i, v := range vlist {
		v = strings.Trim(v, " ")
		if v != "" {
			if u, err := url.Parse(v); err != nil {
				log.Println("invalid vhost", v)
				return
			} else {
				key := u.Host
				if key == "" {
					key = u.Path
				}
				if key == "" {
					key = v
				}
				if _, err := url.Parse(rlist[i]); err != nil {
					log.Println("invalid rhost", rlist[i])
					return
				}
				hosts[strings.ToLower(key)] = strings.ToLower(rlist[i])
			}
		}
	}
	if len(hosts) == 0 {
		log.Println("missing virutal/real hosts lists")
		return
	} else {
		log.Println(hosts)
	}

	// client id seed
	clientId := 0

	for {
		// track pending clients
		clientsPending := 0
		// chan to learn clients ending
		cltEndCh := make(chan int, HUB_RSP_QUEUE_LEN)
		// connect to frontend
		if c, err := dial_hub(*fe_url, *limit); err == nil {
			// prepare chans for hub reader/writer,
			// resources clean up assigment is:
			// - hub reader shall close the hubReqCh
			// - main loop shall close hubRspCh after hubReader dies and
			//   all outgoing clients are done
			// - hub writer should close the underlying conn
			hubReqCh := make(chan *HubRequest, HUB_REQ_QUEUE_LEN)
			hubRspCh := make(chan *http.Response, HUB_RSP_QUEUE_LEN)
			go hubReader(c, hubReqCh)
			go hubWriter(c, hubRspCh)

			proxying := true

			// main forwarding loop
			for proxying {
				select {
				case req, ok := <-hubReqCh:
					if ok {
						// check req to find proper web server
						if srv, ok := hosts[req.req.Host]; ok {
							// start a web client for each req
							clientId += 1
							clientsPending += 1
							go webClient(clientId, req, srv, hubRspCh, cltEndCh)
						} else {
							// no need to start web client
							log.Println("no server for", req.req.Host)
							hubRspCh <- webswitch.QuickResponse(http.StatusNotFound,
								req.req)
						}
					} else {
						// hub reader exited, we need stop looping
						log.Println("hub reader closed")
						hubReqCh = nil
						proxying = false
						break
					}
				case id := <-cltEndCh:
					if clientsPending > 0 {
						clientsPending -= 1
					}
					log.Printf("clnt#%d done, %d pending", id, clientsPending)
				}
			}

			// reader must have closed already, let's wait for pending clients
			for clientsPending > 0 {
				id := <-cltEndCh
				log.Printf("client %d done, %d pending\n", id, clientsPending)
				if clientsPending > 0 {
					clientsPending -= 1
				}
			}
			close(cltEndCh)

			// now safe to close writer
			log.Println("closing hub writer")
			close(hubRspCh)
			hubRspCh = nil
		}
		// sleep for redial later
		log.Println("sleep", *retry_wait, "seconds before redial...")
		time.Sleep(time.Second * time.Duration(*retry_wait))
	}
}
