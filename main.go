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
		slog.Error(err.Error())
		return
	}

	toTCPConn := toConn.(*net.TCPConn)

	if p.keepAlive {
		if err := fromTCPConn.SetKeepAlive(true); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlive failed: %s", err))
		}
		if err := fromTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlivePeriod failed: %s", err))
		}
		if err := toTCPConn.SetKeepAlive(true); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlive failed: %s", err))
		}
		if err := toTCPConn.SetKeepAlivePeriod(p.keepAliveTime); err != nil {
			slog.Warn(fmt.Sprintf("Set KeepAlivePeriod failed: %s", err))
		}
	}

	slog.Debug(fmt.Sprintf("established connection: %s", toConn.RemoteAddr()))
	defer slog.Debug(fmt.Sprintf("association lost: %s<->%s", conn.RemoteAddr(), toConn.RemoteAddr()))

	if _, _, err = bidirectCopy(fromTCPConn, toTCPConn); err != nil {
		slog.Debug(err.Error())
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
	to            string
	verbose       bool
	help          bool
}

func main() {
	opts := runtimeOptions{}
	pflag.StringVarP(&opts.listen, "listen", "l", "", "listen on this addr:port")
	pflag.StringVarP(&opts.to, "to", "t", "", "specify address mapping to")
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
	if opts.listen == "" || opts.to == "" {
		slog.Error("no address mapping specified; specify --listen and --to")
		os.Exit(1)
	}

	slog.Info(fmt.Sprintf("tcp proxy listening on '%s'; proxying to '%s'", opts.listen, opts.to))
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
			slog.Error(err.Error())
			os.Exit(1)
		}
	}()

	slog.Info("started rumpelsepp's rtcp server")
	wg.Wait()
	slog.Error("proxy terminated")
	os.Exit(1)
}
