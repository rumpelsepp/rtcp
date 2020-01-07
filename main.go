package main

import (
	"net"
	"os"

	"git.sr.ht/~rumpelsepp/helpers"
	"git.sr.ht/~rumpelsepp/rlog"
	"git.sr.ht/~sircmpwn/getopt"
)

type runtimeOptions struct {
	from    string
	to      string
	verbose bool
	help    bool
}

func handleClient(conn net.Conn, dst string) {
	fromConn := conn
	toConn, err := net.Dial("tcp", dst)
	if err != nil {
		rlog.Warning(err)
		return
	}

	rlog.Debugf("established connection to: %s", dst)

	if _, _, err = helpers.BidirectCopy(fromConn, toConn); err != nil {
		rlog.Warning(err)
	}
}

func main() {
	opts := runtimeOptions{}
	getopt.StringVar(&opts.from, "f", "", "specify address mapping from")
	getopt.StringVar(&opts.to, "t", "", "specify address mapping to")
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

	if opts.from == "" || opts.to == "" {
		rlog.Crit("no address mapping specified")
	}

	ln, err := net.Listen("tcp", opts.from)
	if err != nil {
		rlog.Crit(err)
	}

	rlog.Infof("started rumpelsepp's rtcp server")
	rlog.Infof("serving from '%s' to '%s'", opts.from, opts.to)

	for {
		conn, err := ln.Accept()
		if err != nil {
			rlog.Warning(err)
			continue
		}

		rlog.Debugf("got connection from: %v", opts.from)

		go handleClient(conn, opts.to)
	}
}
