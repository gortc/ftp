// This binary is an example of virtual ftp.
package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"

	"gortc.io/ftp"
	"gortc.io/ftp/vd"
)

type printProxy struct{}

func (printProxy) ProxyFrom(r io.Reader, offset int64) (int64, error) {
	log.Printf("[proxy]: writing (offset: %d)", offset)
	n, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		return n, err
	}
	log.Printf("[proxy]: wrote %d (offset: %d)", n, offset)
	return n, nil
}

func main() {
	var (
		port = flag.Int("port", 2121, "Port")
		host = flag.String("host", "localhost", "Host")
	)
	flag.Parse()

	factory := &vd.Factory{
		Proxy: printProxy{},
	}

	opts := &ftp.ServerOpts{
		Factory:  factory,
		Port:     *port,
		Hostname: *host,
		Auth:     ftp.NoAuth,
	}

	log.Printf("Starting virtual ftp server on %v:%v", opts.Hostname, opts.Port)
	server := ftp.NewServer(opts)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
