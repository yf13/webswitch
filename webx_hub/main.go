// Copyright 2015 The WebSwitch authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"github.com/yf13/webswitch"
	"log"
	"net/http"
	"strings"
)

// version of frontend switch
const APP_VERSION = "0.1"

// command line options
var (
	cert_file   = flag.String("cert", "", "public cert file (.pem) w/ CA and SANs")
	key_file    = flag.String("key", "", "private key file (.pem)")
	hub_path    = flag.String("path", webswitch.HUB_RESOURCE_NAME, "hub resource path.")
	http_ports  = flag.String("http_ports", ":8080", "comma separated ports for http clients.")
	https_ports = flag.String("https_ports", ":8443", "comma separated ports for https clients.")
	plug_port   = flag.String("plug", ":8081", "port for plugs.")
	//auth_plugs  = flag.Bool("auth", false, "whether to challenge plugs")
)

// program entrance
func main() {

	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.Ldate)
	flag.Parse()

	log.Println("version:", APP_VERSION)

	secured := len(*cert_file) > 0 && len(*key_file) > 0

	// check key files

	// start client listeners with default server mux
	http.HandleFunc("/", handleClient)
	if len(*http_ports) > 0 {
		for _, port := range strings.Split(*http_ports, ",") {
			log.Println("http port: ", port)
			go func() {
				log.Fatal(
					http.ListenAndServe(port, nil))
			}()
		}
	}
	if secured && len(*https_ports) > 0 {
		for _, port := range strings.Split(*https_ports, ",") {
			log.Println("https port: ", port)
			go func() {
				log.Fatal(
					http.ListenAndServeTLS(port, *cert_file, *key_file, nil))
			}()
		}
	}

	// start the hub
	go hub.run()

	// start plug port on its own server mux, special case
	// if plug port same as http/https port, listen will fail now.
	smuxPlug := http.NewServeMux()
	smuxPlug.HandleFunc(*hub_path, handlePlug)
	//http.HandleFunc(*hub_path, handlePlug)
	if !secured {
		log.Println("insecure plug: ", *plug_port+*hub_path)
		log.Fatal("ListenAndServe: ", http.ListenAndServe(*plug_port, smuxPlug))
	} else {
		log.Println("secure plug: ", *plug_port+*hub_path)
		log.Fatal("ListenAndServeTLS: ", http.ListenAndServeTLS(*plug_port,
			*cert_file, *key_file, smuxPlug))
	}
}
