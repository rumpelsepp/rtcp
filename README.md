<!--
SPDX-FileCopyrightText: Stefan Tatschner

SPDX-License-Identifier: MIT
-->

# rtcp
## TCP Proxy

`rtcp` is a **very simple** reverse TCP and SOCKS proxy.
Its only purpose is to accept a TCP connection, dial out to a remote host:port and forward everything.
Port forwarding setups are trivial:

```
$ rtcp -l :8000 -t 1.1.1.1:80
```

This opens port 8000 on all interfaces and forwards everything to 1.1.1.1:80.
Multiple connections are supported.
