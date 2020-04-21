package main

import (
	"net"
	"os"
	"sync"
	"time"

	"git.sr.ht/~rumpelsepp/helpers"
	"git.sr.ht/~rumpelsepp/rlog"
	"git.sr.ht/~rumpelsepp/socks5"
	"git.sr.ht/~sircmpwn/getopt"
)

type tcpProxy struct {
	listen        string
	dst           string
	keepAlive     bool
	keepAliveTime time.Duration
}

func (p *tcpProxy) handleClient(conn *net.TCPConn) {
	fromTCPConn := conn
	toConn, err := net.Dial("tcp", p.dst)
	if err != nil {
		rlog.Warning(err)
		return
	}

	toTCPConn := toConn.(*net.TCPConn)

	if p.keepAlive {
		if err := fromTCPConn.SetKeepAlive(true); err != nil {
			rlog.Warningf("Set KeepAlive failed: %s", err)
		}
		if err := fromTCPConn.SetKeepAlivePeriod(time.Duration(p.keepAliveTime) * time.Second); err != nil {
			rlog.Warning("Set KeepAlivePeriod failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlive(true); err != nil {
			rlog.Warningf("Set KeepAlive failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			rlog.Warning("Set KeepAlivePeriod failed: %s", err)
		}
	}

	rlog.Debugf("established connection: %s", toConn.RemoteAddr())
	defer rlog.Debugf("association lost: %s<->%s", conn.RemoteAddr(), toConn.RemoteAddr())

	if _, _, err = helpers.BidirectCopy(fromTCPConn, toTCPConn); err != nil {
		rlog.Debug(err)
	}
}

func (p *tcpProxy) Serve() error {
	ln, err := net.Listen("tcp", p.listen)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			rlog.Warning(err)
			continue
		}

		rlog.Debugf("got connection: %s", conn.RemoteAddr())
		go p.handleClient(conn.(*net.TCPConn))
	}
}

type runtimeOptions struct {
	listen        string
	keepAlive     bool
	keepAliveTime int
	to            string
	socks         string
	verbose       bool
	help          bool
}

func main() {
	opts := runtimeOptions{}
	getopt.StringVar(&opts.listen, "l", "", "listen on this addr:port")
	getopt.StringVar(&opts.to, "t", "", "specify address mapping to")
	getopt.BoolVar(&opts.keepAlive, "a", false, "enable tcp keepalive probes")
	getopt.IntVar(&opts.keepAliveTime, "k", 25, "specify keepalive time in seconds")
	getopt.StringVar(&opts.socks, "s", "", "enable a socks5 listener on addr:port")
	getopt.BoolVar(&opts.verbose, "v", false, "enable debugging output")
	getopt.BoolVar(&opts.help, "h", false, "show this page and exit")

	err := getopt.Parse()
	if err != nil {
		rlog.Crit(err)
	}

	if opts.help {
		getopt.Usage()
		os.Exit(0)
	}

	if opts.verbose {
		rlog.SetLogLevel(rlog.DEBUG)
	}

	var wg sync.WaitGroup
	if opts.listen != "" {
		if opts.listen == "" || opts.to == "" {
			rlog.Crit("no address mapping specified")
		}

		rlog.Infof("tcp proxy listening on '%s'; proxying to '%s'", opts.listen, opts.to)
		wg.Add(1)
		go func() {
			proxy := tcpProxy{
				listen:        opts.listen,
				dst:           opts.to,
				keepAlive:     opts.keepAlive,
				keepAliveTime: time.Duration(opts.keepAliveTime) * time.Second,
			}
			err := proxy.Serve()
			rlog.Crit(err)
			wg.Done()
		}()
	}
	if opts.socks != "" {
		rlog.Infof("socks5 proxy listening on '%s'", opts.socks)
		wg.Add(1)
		go func() {
			proxy := socks5.NewServer(opts.socks)
			err := proxy.Serve()
			rlog.Crit(err)
			wg.Done()
		}()

	}
	rlog.Infof("started rumpelsepp's rtcp server")
	wg.Wait()
}
