// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ftp

import "io"

// DriverFactory is a driver factory to create driver. For each client that connects to the server, a new FTPDriver is required.
// Create an implementation if this interface and provide it to FTPServer.
type DriverFactory interface {
	NewDriver() (Driver, error)
}

// Driver is an interface that you will create an implementation that speaks to your
// chosen persistence layer. graval will create a new instance of your
// driver for each client that connects and delegate to it as required.
type Driver interface {
	// Init init
	Init(*Conn)

	// params  - a file path
	// returns - a time indicating when the requested path was last modified
	//         - an error if the file doesn't exist or the user lacks
	//           permissions
	Stat(path string) (FileInfo, error)

	// params  - path
	// returns - true if the current user is permitted to change to the
	//           requested path
	ChangeDir(path string) error

	// params  - path, function on file or subdir found
	// returns - error
	//           path
	ListDir(string, func(FileInfo) error) error

	// params  - path
	// returns - nil if the directory was deleted or any error encountered
	DeleteDir(path string) error

	// params  - path
	// returns - nil if the file was deleted or any error encountered
	DeleteFile(path string) error

	// params  - from_path, to_path
	// returns - nil if the file was renamed or any error encountered
	Rename(from string, to string) error

	// params  - path
	// returns - nil if the new directory was created or any error encountered
	MakeDir(path string) error

	// params  - path
	// returns - a string containing the file data to send to the client
	GetFile(name string, offset int64) (int64, io.ReadCloser, error)

	// params  - destination path, an io.Reader containing the file data
	// returns - the number of bytes writen and the first error encountered while writing, if any.
	PutFile(name string, offset int64, r io.Reader, append bool) (int64, error)

	Abort() error

	Close() error
}
