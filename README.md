<!--
SPDX-FileCopyrightText: Stefan Tatschner

SPDX-License-Identifier: MIT
-->

# rtcp (Relay TCP)

`rtcp` is a very simple TCP relay.
Its only purpose is to accept TCP connections, dial out to a remote host:port and forward everything.

## Build

```
$ make
```

## Run

This opens port 8000 on all interfaces and forwards everything to 1.1.1.1:80.
Concurrent connections, hostnames, and IPv4/IPv6 addresses are supported.

```
$ rtcp -l :8000 -t 1.1.1.1:80
```

For better readability, long arguments are supported:

```
$ rtcp --listen :8000 --target 1.1.1.1:80
```

TCP keepalive probes can be enabled as well:

```
$ rtcp --keep-alive --listen :8000 --target 1.1.1.1:80
```
