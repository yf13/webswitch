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

Once the intranet site is plugged into the **hub**, they can be accessed like normal Internet web sites as shown in below diagram: 

```sequence
participant Internet\nvisitor as Visitor
participant WebSwitch\n Hub as Hub
participant WebSwitch\nPlug as Plug 
participant Intranet\nsite as Site
note left of Hub: opening\nchannels
Plug->Hub: open
Hub->Plug: plug channel
note right of Visitor: serving\nvisitors
Visitor->Hub: web request
Hub-->Plug: forwarded request
Plug->Site: web request
Site->Plug: web response
Plug-->Hub: forwarded response
Hub->Visitor: web response
```

Installation 
-------------

WebSwitch is written in [Go](http://golang.org) programming language, thus you can install it from source with:

	go get github.com/yf13/webswitch

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


