package main

import (
	"net"
	"os"
	"time"

	"git.sr.ht/~rumpelsepp/helpers"
	"git.sr.ht/~rumpelsepp/rlog"
	"git.sr.ht/~sircmpwn/getopt"
)

func handleClient(conn *net.TCPConn, dst string, keepAlive bool, keepAliveTime int) {
	fromTCPConn := conn
	toConn, err := net.Dial("tcp", dst)
	if err != nil {
		rlog.Warning(err)
		return
	}

	toTCPConn := toConn.(*net.TCPConn)

	if keepAlive {
		if err := fromTCPConn.SetKeepAlive(true); err != nil {
			rlog.Warningf("Set KeepAlive failed: %s", err)
		}
		if err := fromTCPConn.SetKeepAlivePeriod(time.Duration(keepAliveTime) * time.Second); err != nil {
			rlog.Warning("Set KeepAlivePeriod failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlive(true); err != nil {
			rlog.Warningf("Set KeepAlive failed: %s", err)
		}
		if err := toTCPConn.SetKeepAlivePeriod(time.Duration(keepAliveTime) * time.Second); err != nil {
			rlog.Warning("Set KeepAlivePeriod failed: %s", err)
		}
	}

	rlog.Debugf("established connection: %s", toConn.RemoteAddr())
	defer rlog.Debugf("association lost: %s<->%s", conn.RemoteAddr(), toConn.RemoteAddr())

	if _, _, err = helpers.BidirectCopy(fromTCPConn, toTCPConn); err != nil {
		rlog.Debug(err)
	}
}

type runtimeOptions struct {
	listen        string
	keepAlive     bool
	keepAliveTime int
	to            string
	verbose       bool
	help          bool
}

func main() {
	opts := runtimeOptions{}
	getopt.StringVar(&opts.listen, "l", ":8000", "listen on this addr:port")
	getopt.StringVar(&opts.to, "t", "", "specify address mapping to")
	getopt.BoolVar(&opts.keepAlive, "a", false, "enable tcp keepalive probes")
	getopt.IntVar(&opts.keepAliveTime, "k", 25, "specify keepalive time in seconds")
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

	if opts.listen == "" || opts.to == "" {
		rlog.Crit("no address mapping specified")
	}

	ln, err := net.Listen("tcp", opts.listen)
	if err != nil {
		rlog.Crit(err)
	}

	rlog.Infof("started rumpelsepp's rtcp server")
	rlog.Infof("listening on '%s'; proxying to '%s'", opts.listen, opts.to)

	for {
		conn, err := ln.Accept()
		if err != nil {
			rlog.Warning(err)
			continue
		}

		rlog.Debugf("got connection: %s", conn.RemoteAddr())
		go handleClient(conn.(*net.TCPConn), opts.to, opts.keepAlive, opts.keepAliveTime)
	}
}
