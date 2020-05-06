# rtcp
## TCP Proxy

`rtcp` is a **very simple** reverse TCP and SOCKS proxy.
Its only purpose is to accept a TCP connection, dial out to a remote host:port and forward everything.
Port forwarding setups are trivial:

```
$ rtcp -l :8000 -t 1.1.1.1:80
```

This opens port 8000 on all interfaces and forwards everything to 1.1.1.1:80.

## SOCKS5 proxy

`rtcp` also supports SOCKS5 for dynamic portforwarding.
Username and Password based authentication as available.
Check the help with `-h`.

```
$ rtcp -s ":1080"
```

This opens port 1080 and it expects SOCKS5 traffic there via e.g.:

```
$ all_proxy=socks5://[::1]:1080 curl -L google.de
```
