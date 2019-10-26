// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp_test

import (
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"gortc.io/ftp"
	"gortc.io/ftp/fd"

	jftp "github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
)

func runServer(t *testing.T, execute func()) {
	os.MkdirAll("./testdata", os.ModePerm)

	var perm = ftp.NewSimplePerm("test", "test")
	opt := &ftp.ServerOpts{
		Name: "test ftpd",
		Factory: &fd.Factory{
			RootPath: "./testdata",
			Perm:     perm,
		},
		Port: 2121,
		Auth: &ftp.SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		Logger: new(ftp.DiscardLogger),
	}

	s := ftp.NewServer(opt)
	go func() {
		err := s.ListenAndServe()
		assert.EqualError(t, err, ftp.ErrServerClosed.Error())
	}()

	execute()

	assert.NoError(t, s.Shutdown())
}

func TestConnect(t *testing.T) {
	runServer(t, func() {
		// Give server 0.5 seconds to get to the listening state
		timeout := time.NewTimer(time.Millisecond * 500)
		for {
			f, err := jftp.Connect("localhost:2121")
			if err != nil && len(timeout.C) == 0 { // Retry errors
				continue
			}
			assert.NoError(t, err)

			assert.NoError(t, f.Login("admin", "admin"))
			assert.Error(t, f.Login("admin", ""))

			var content = `test`
			assert.NoError(t, f.Stor("server_test.go", strings.NewReader(content)))

			names, err := f.NameList("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 1, len(names))
			assert.EqualValues(t, "server_test.go", names[0])

			bs, err := ioutil.ReadFile("./testdata/server_test.go")
			assert.NoError(t, err)
			assert.EqualValues(t, content, string(bs))

			entries, err := f.List("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 1, len(entries))
			assert.EqualValues(t, "server_test.go", entries[0].Name)
			assert.EqualValues(t, 4, entries[0].Size)
			assert.EqualValues(t, jftp.EntryTypeFile, entries[0].Type)

			curDir, err := f.CurrentDir()
			assert.NoError(t, err)
			assert.EqualValues(t, "/", curDir)

			size, err := f.FileSize("/server_test.go")
			assert.NoError(t, err)
			assert.EqualValues(t, 4, size)

			/*resp, err := f.RetrFrom("/server_test.go", 0)
			assert.NoError(t, err)
			var buf []byte
			l, err := resp.Read(buf)
			assert.NoError(t, err)
			assert.EqualValues(t, 4, l)
			assert.EqualValues(t, 4, len(buf))
			assert.EqualValues(t, content, string(buf))*/

			err = f.Rename("/server_test.go", "/ftp.test.go")
			assert.NoError(t, err)

			err = f.MakeDir("/src")
			assert.NoError(t, err)

			err = f.Delete("/ftp.test.go")
			assert.NoError(t, err)

			err = f.RemoveDir("/src")
			assert.NoError(t, err)

			err = f.Quit()
			assert.NoError(t, err)

			break
		}
	})
}

func TestServe(t *testing.T) {
	os.MkdirAll("./testdata", os.ModePerm)

	var perm = ftp.NewSimplePerm("test", "test")

	// Server options without hostname or port
	opt := &ftp.ServerOpts{
		Name: "test ftpd",
		Factory: &fd.Factory{
			RootPath: "./testdata",
			Perm:     perm,
		},
		Auth: &ftp.SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		Logger: new(ftp.DiscardLogger),
	}

	// Start the listener
	l, err := net.Listen("tcp", ":2121")
	assert.NoError(t, err)

	// Start the server using the listener
	s := ftp.NewServer(opt)
	go func() {
		err := s.Serve(l)
		assert.EqualError(t, err, ftp.ErrServerClosed.Error())
	}()

	// Give server 0.5 seconds to get to the listening state
	timeout := time.NewTimer(time.Millisecond * 500)
	for {
		f, err := jftp.Connect("localhost:2121")
		if err != nil && len(timeout.C) == 0 { // Retry errors
			continue
		}
		assert.NoError(t, err)

		assert.NoError(t, f.Login("admin", "admin"))
		assert.Error(t, f.Login("admin", ""))

		err = f.Quit()
		assert.NoError(t, err)
		break
	}

	assert.NoError(t, s.Shutdown())
}
