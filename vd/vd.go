// Package vd provides a virtual FTP driver with single file.
// File name is "output" and it already exists with zero size.
package vd

import (
	"errors"
	"io"
	"os"
	"time"

	"gortc.io/ftp"
)

var errNotImplemented = errors.New("not implemented")

type fileInfo struct {
	name string
	size int64
}

func (f fileInfo) Name() string       { return f.name }
func (f fileInfo) Size() int64        { return f.size }
func (f fileInfo) Mode() os.FileMode  { return 0666 }
func (f fileInfo) ModTime() time.Time { return time.Time{} }
func (f fileInfo) IsDir() bool        { return false }
func (f fileInfo) Owner() string      { return "owner" }
func (f fileInfo) Group() string      { return "group" }
func (f fileInfo) Sys() interface{}   { return nil }

type Proxy interface {
	// ProxyFrom copies all data from reader, returning bytes read.
	ProxyFrom(r io.Reader, offset int64) (int64, error)
	// Close concludes that there will be no writes.
	Close() error
}

type Driver struct {
	proxy Proxy
	size  int64
}

const fileName = "output"

func (d Driver) Name() string { return fileName }

func (d Driver) Stat(name string) (ftp.FileInfo, error) {
	f := fileInfo{
		name: fileName,
		size: d.size,
	}
	return f, nil
}

func (d Driver) PutFile(name string, offset int64, r io.Reader, append bool) (int64, error) {
	if name != "/"+fileName {
		return 0, errors.New("unexpected file name")
	}
	n, err := d.proxy.ProxyFrom(r, offset)
	d.size += n // TODO: Totally wrong. Fix if needed.
	return n, err
}

func (d Driver) Abort() error { return d.proxy.Close() }

func (d Driver) Init(*ftp.Conn)                                      {}
func (d Driver) ChangeDir(string) error                              { return errNotImplemented }
func (d Driver) ListDir(string, func(ftp.FileInfo) error) error      { return errNotImplemented }
func (d Driver) DeleteDir(string) error                              { return errNotImplemented }
func (d Driver) DeleteFile(string) error                             { return errNotImplemented }
func (d Driver) Rename(string, string) error                         { return errNotImplemented }
func (d Driver) MakeDir(string) error                                { return errNotImplemented }
func (d Driver) GetFile(string, int64) (int64, io.ReadCloser, error) { return 0, nil, errNotImplemented }

type Factory struct {
	Proxy Proxy
}

func (f *Factory) NewDriver() (ftp.Driver, error) {
	return &Driver{proxy: f.Proxy}, nil
}
