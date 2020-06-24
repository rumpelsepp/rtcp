package main

import (
	"net"
	"os"
	"sync"
	"time"

	"git.sr.ht/~rumpelsepp/helpers"
	"git.sr.ht/~rumpelsepp/socks5"
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/Fraunhofer-AISEC/penlog"
)

var logger = penlog.NewLogger("", os.Stderr)

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
		logger.LogWarning(err)
		return
	}

	toTCPConn := toConn.(*net.TCPConn)

	if p.keepAlive {
		if err := fromTCPConn.SetKeepAlive(true); err != nil {
			logger.LogWarningf("Set KeepAlive failed: %s", err)
		}
		if err := fromTCPConn.SetKeepAlivePeriod(time.Duration(p.keepAliveTime) * time.Second); err != nil {
			logger.LogWarning("Set KeepAlivePeriod failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlive(true); err != nil {
			logger.LogWarningf("Set KeepAlive failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			logger.LogWarning("Set KeepAlivePeriod failed: %s", err)
		}
	}

	logger.LogDebugf("established connection: %s", toConn.RemoteAddr())
	defer logger.LogDebugf("association lost: %s<->%s", conn.RemoteAddr(), toConn.RemoteAddr())

	if _, _, err = helpers.BidirectCopy(fromTCPConn, toTCPConn); err != nil {
		logger.LogDebug(err)
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
			logger.LogWarning(err)
			continue
		}

		logger.LogDebugf("got connection: %s", conn.RemoteAddr())
		go p.handleClient(conn.(*net.TCPConn))
	}
}

type runtimeOptions struct {
	listen        string
	keepAlive     bool
	keepAliveTime int
	to            string
	socks         string
	username      string
	password      string
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
	getopt.StringVar(&opts.username, "u", "", "optional username for SOCKS server")
	getopt.StringVar(&opts.password, "p", "", "optional password for SOCKS server")
	getopt.BoolVar(&opts.verbose, "v", false, "enable debugging output")
	getopt.BoolVar(&opts.help, "h", false, "show this page and exit")

	err := getopt.Parse()
	if err != nil {
		logger.LogError(err)
		os.Exit(1)
	}

	if opts.help {
		getopt.Usage()
		os.Exit(0)
	}

	logger.SetColors(true)
	if opts.verbose {
		logger.SetLogLevel(penlog.PrioDebug)
	} else {
		logger.SetLogLevel(penlog.PrioInfo)
	}

	var wg sync.WaitGroup
	if opts.listen != "" {
		if opts.listen == "" || opts.to == "" {
			logger.LogError("no address mapping specified")
			os.Exit(1)
		}

		logger.LogInfof("tcp proxy listening on '%s'; proxying to '%s'", opts.listen, opts.to)
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy := tcpProxy{
				listen:        opts.listen,
				dst:           opts.to,
				keepAlive:     opts.keepAlive,
				keepAliveTime: time.Duration(opts.keepAliveTime) * time.Second,
			}
			if err := proxy.Serve(); err != nil {
				logger.LogError(err)
				os.Exit(1)
			}
		}()
	}
	if opts.socks != "" {
		logger.LogInfof("socks5 proxy listening on '%s'", opts.socks)
		wg.Add(1)
		go func() {
			defer wg.Done()
			proxy := socks5.NewServer(opts.socks, opts.username, opts.password)
			if err := proxy.Serve(); err != nil {
				logger.LogError(err)
				os.Exit(1)
			}
		}()

	}
	logger.LogInfof("started rumpelsepp's rtcp server")
	wg.Wait()
	logger.LogError("proxy terminated, did you provide a config?")
}
