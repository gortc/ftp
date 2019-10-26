package vd

import (
	"bytes"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	client "github.com/jlaffaye/ftp"

	"gortc.io/ftp"
)

// testLogger is logger that prints to testing log as helper.
type testLogger struct {
	t *testing.T
}

func (log *testLogger) Print(sessionId string, message interface{}) {
	log.t.Helper()
	log.t.Log(message)
}
func (log *testLogger) Printf(sessionId string, format string, v ...interface{}) {
	log.t.Helper()
	log.t.Logf(format, v...)
}
func (log *testLogger) PrintCommand(sessionId string, command string, params string) {
	log.t.Helper()
	log.t.Logf("> %s %s", command, params)
}
func (log *testLogger) PrintResponse(sessionId string, code int, message string) {
	log.t.Helper()
	log.t.Logf("< %d %s", code, message)
}

type testProxy struct {
	f     func(r io.Reader, offset int64) (int64, error)
	abort func() error
}

func (p *testProxy) ProxyFrom(r io.Reader, offset int64) (int64, error) { return p.f(r, offset) }
func (p *testProxy) Close() error                                       { return p.abort() }

func TestDriver(t *testing.T) {
	proxy := &testProxy{
		f: func(r io.Reader, offset int64) (i int64, e error) {
			t.Error("should not be called")
			return 0, nil
		},
		abort: func() error {
			t.Error("should not be called")
			return nil
		},
	}
	factory := &Factory{
		Proxy: proxy,
	}

	opts := &ftp.ServerOpts{
		Factory:  factory,
		Hostname: "127.0.0.1",
		Auth:     ftp.NoAuth,
		Logger:   &testLogger{t: t},
	}

	log.Printf("Starting virtual ftp server on %v:%v", opts.Hostname, opts.Port)
	server := ftp.NewServer(opts)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	stopped := make(chan struct{})
	go func() {
		_ = server.Serve(l)
		stopped <- struct{}{}
	}()

	conn, err := client.Connect(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Login("1234", "569"); err != nil {
		t.Fatalf("failed to login: %v", err)
	}
	dir, err := conn.CurrentDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/" {
		t.Error("invalid current dir")
	}

	// From start.
	proxy.f = func(r io.Reader, offset int64) (i int64, e error) {
		if offset != 0 {
			t.Error("unexpected offset")
		}
		buf := new(bytes.Buffer)
		n, err := io.Copy(buf, r)
		if err != nil {
			t.Error(err)
		}
		if buf.String() != "foo" {
			t.Error("unexpected content")
		}
		return n, nil
	}
	if err = conn.StorFrom("/output", strings.NewReader("foo"), 0); err != nil {
		t.Fatal(err)
	}

	// Starting from 2.
	proxy.f = func(r io.Reader, offset int64) (i int64, e error) {
		if offset != 2 {
			t.Error("unexpected offset")
		}
		buf := new(bytes.Buffer)
		n, err := io.Copy(buf, r)
		if err != nil {
			t.Error(err)
		}
		if buf.String() != "bar" {
			t.Error("unexpected content")
		}
		return n, nil
	}
	if err = conn.StorFrom("/output", strings.NewReader("bar"), 2); err != nil {
		t.Fatal(err)
	}

	// Call on /not-found file should fail.
	proxy.f = func(r io.Reader, offset int64) (i int64, e error) {
		t.Error("should not be called")
		return 0, nil
	}
	if err = conn.StorFrom("/not-found", strings.NewReader("baz"), 1000); err == nil {
		t.Error("should fail")
	}
	if err := server.Shutdown(); err != nil {
		t.Error(err)
	}
	select {
	case <-stopped:
		// OK
	case <-time.After(time.Second * 10):
		t.Fatal("Timed out")
	}
}
