// SPDX-FileCopyrightText: Stefan Tatschner
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

func bidirectCopy(left io.ReadWriteCloser, right io.ReadWriteCloser) (int, int, error) {
	var (
		n1   = 0
		n2   = 0
		err  error
		err1 error
		err2 error
		wg   sync.WaitGroup
	)

	wg.Add(2)

	go func() {
		if n, err := io.Copy(right, left); err != nil {
			err1 = err
		} else {
			n1 = int(n)
		}

		right.Close()
		wg.Done()
	}()

	go func() {
		if n, err := io.Copy(left, right); err != nil {
			err2 = err
		} else {
			n2 = int(n)
		}

		left.Close()
		wg.Done()
	}()

	wg.Wait()

	if err1 != nil && err2 != nil {
		err = fmt.Errorf("both copier failed; left: %s; right: %s", err1, err2)
	} else {
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		}
	}

	return n1, n2, err
}

type tcpRelay struct {
	listen        string
	target        string
	keepAlive     bool
	keepAliveTime time.Duration
}

func (p *tcpRelay) handleClient(conn *net.TCPConn) {
	fromTCPConn := conn
	targetConn, err := net.Dial("tcp", p.target)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	targetTCPConn := targetConn.(*net.TCPConn)

	if p.keepAlive {
		if err := fromTCPConn.SetKeepAlive(true); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlive failed: %s", err))
		}
		if err := fromTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlivePeriod failed: %s", err))
		}
		if err := targetTCPConn.SetKeepAlive(true); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlive failed: %s", err))
		}
		if err := targetTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlivePeriod failed: %s", err))
		}
	}

	slog.Debug(fmt.Sprintf("established connection: %s", targetConn.RemoteAddr()))
	defer slog.Debug(fmt.Sprintf("association lost: %s<->%s", conn.RemoteAddr(), targetConn.RemoteAddr()))

	if _, _, err = bidirectCopy(fromTCPConn, targetTCPConn); err != nil {
		slog.Debug(err.Error())
	}
}

func (p *tcpRelay) Serve() error {
	ln, err := net.Listen("tcp", p.listen)
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Warn(err.Error())
			continue
		}

		slog.Debug(fmt.Sprintf("got connection: %s", conn.RemoteAddr()))
		go p.handleClient(conn.(*net.TCPConn))
	}
}

type runtimeOptions struct {
	listen        string
	keepAlive     bool
	keepAliveTime time.Duration
	target        string
	verbose       bool
	help          bool
}

func main() {
	opts := runtimeOptions{}
	pflag.StringVarP(&opts.listen, "listen", "l", "", "listen on this addr:port; hostnames are allowed")
	pflag.StringVarP(&opts.target, "target", "t", "", "specify relay target")
	pflag.BoolVar(&opts.keepAlive, "keep-alive", false, "enable tcp keepalive probes")
	pflag.DurationVar(&opts.keepAliveTime, "keep-alive-time", 25*time.Second, "specify keepalive time in seconds")
	pflag.BoolVarP(&opts.verbose, "verbose", "v", false, "enable debugging output")

	pflag.Parse()

	var level slog.Level
	if opts.verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	var wg sync.WaitGroup
	if opts.listen == "" || opts.target == "" {
		slog.Error("no address mapping specified; specify --listen and --target")
		os.Exit(1)
	}

	slog.Info(fmt.Sprintf("tcp relay listening on '%s'; forwarding to '%s'", opts.listen, opts.target))
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy := tcpRelay{
			listen:        opts.listen,
			target:        opts.target,
			keepAlive:     opts.keepAlive,
			keepAliveTime: time.Duration(opts.keepAliveTime) * time.Second,
		}
		if err := proxy.Serve(); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}()

	slog.Info("started rumpelsepp's rtcp server")
	wg.Wait()
	slog.Error("proxy terminated")
	os.Exit(1)
}
