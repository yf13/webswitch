Welcome to WebSwitch
===================

WebSwitch helps to publish web sites running at intranet to Internet visitors easily. 

WebSwitch includes two simple programs: a **hub** and a **plug**. 

The **hub**:

 > - runs as your Internet web server for Internet visitors to access your published web sites. 
 > - accepts incoming connections from **plugs** to get your web sites published.

The **plug**: 

> 
> - runs in your intranet as client of your intranet site.
> - uses outgoing TCP connections to the **hub** to publish your intranet site.

How it works
-------------

The WebSwitch **hub** should be running as a web server on an Internet host so that both visitors 
and plugs can reach it at any time. 

Then the WebSwitch **plug** should be started on proper intranet host so that it can access both
the **hub** and the intranet web site. Once started, the **plug** will connect to the **hub** to
advertise the sites to be published.

Then visitors can access the DNS the sites which will all point to the **hub**. The **hub** will forward visitors' requests to intranet sites through the **plugs** as shown in this ![diagram](/how-it-works.png). 


Installation 
-------------

WebSwitch is written in [Go](http://golang.org) programming language, thus you can install it from source with:

   ```
	go get github.com/yf13/webswitch
   ```

To build the binaries from source, you can use the "go instrall" commnand like: 

   ```
   cd $GOPATH/github.com/yf13/webswtich/webx_plug; go install
   cd $GOPATH/github.com/yf13/webswitch/webx_hub; go install
   ```

The output executable binaries can be found at $GOPATH/bin, such as 

   ```
   $ ls -l $GOPATH/bin/webx
   -rwxr-xr-x 1 u u 7214992 Jun 11 19:35 /home/user/go/bin/webx_hub*
   -rwxr-xr-x 1 u u 7198424 Jun 11 19:35 /home/user/go/bin/webx_plug*
   ```

Note that current version also requires [WebSocket] (https://github.com/gorilla/websocket).

Simple Usage
---------

- On the public server pointed by www.example.com, start the **hub** like:

```
   webx_hub
```

- On the intranet where intranet site listens at localhost:8080, start the **plug** using:

```
   webx_plug -hub ws://www.example.com:8081/_webx -hosts www.example.com:8080 -rhosts http://localhost:8080

```

Then from a client host use a browser or cURL tool to access http://www.example.com:8080.

Note that since the intranet site listens on localhost only, it can't be reached from the client host directly. However with webswitch, the site can still be published on the **hub** as shown above.

Serious Usage
-------------

For serious usage, secured plug connections may be preferred based on your network toplogy. proper certificates should be supplied to both programs, see coomand  options for more details.

Command Options
--------

The **hub** program accepts the following options:

```
  -cert string
      public cert file (.pem) w/ CA and SANs
  -http_ports string
      comma separated ports for http clients. (default ":8080")
  -https_ports string
      comma separated ports for https clients. (default ":8443")
  -key string
      private key file (.pem)
  -path string
      hub resource path. (default "/_webx")
  -plug string
      port for plugs. (default ":8081")
```

The **plug** program accepts the following options:

```
  -ca string
      root CA pem: ca.crt
  -cert string
      plug public signed cert.crt.
  -hosts string
      comma separated virtual hosts (e.g. 'ibm.com:8080,hp.com')
  -hub string
      hub's URL to plug into. (e.g. wss://hub:8443/_webx)
  -key string
      plug private key.pem.
  -limit int
      size limit, 0 is unlimited.
  -retry int
      redial waiting seconds (default 60)
  -rhosts string
      comma separated corresponding real hosts (e.g. 'http://localhost:8081,http://localhost:8082')
```

Use "-h" option to learn the command line options for webx_hub and webx_plug programs.




Limitations
----------------

 - Web sites using WebSocket is not supported yet;
 

License
-----------

WebSwitch is BSD licensed, see the LICENSE file for details.


