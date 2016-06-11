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

Then visitors can access the DNS the sites which will all point to the **hub**. The **hub** will forward visitors' requests to intranet sites through the **plugs** as shown in this [diagram](./how-it-works.png). 


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

Usage
---------



Use "-h" option to learn the command line options for webx_hub and webx_plug programs.


Limitations
----------------

 - Web sites using WebSocket is not supported yet;
 

License
-----------

WebSwitch is BSD licensed, see the LICENSE file for details.


