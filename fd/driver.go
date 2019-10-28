package fd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gortc.io/ftp"
)

type Driver struct {
	RootPath string
	ftp.Perm
}

type FileInfo struct {
	os.FileInfo

	mode  os.FileMode
	owner string
	group string
}

func (f *FileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *FileInfo) Owner() string {
	return f.owner
}

func (f *FileInfo) Group() string {
	return f.group
}

func (d *Driver) realPath(path string) string {
	paths := strings.Split(path, "/")
	return filepath.Join(append([]string{d.RootPath}, paths...)...)
}

func (d *Driver) Init(conn *ftp.Conn) {
	//d.conn = conn
}

func (d *Driver) ChangeDir(path string) error {
	rPath := d.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if f.IsDir() {
		return nil
	}
	return errors.New("not a directory")
}

func (d *Driver) Stat(path string) (ftp.FileInfo, error) {
	basepath := d.realPath(path)
	rPath, err := filepath.Abs(basepath)
	if err != nil {
		return nil, err
	}
	f, err := os.Lstat(rPath)
	if err != nil {
		return nil, err
	}
	mode, err := d.Perm.GetMode(path)
	if err != nil {
		return nil, err
	}
	if f.IsDir() {
		mode |= os.ModeDir
	}
	owner, err := d.Perm.GetOwner(path)
	if err != nil {
		return nil, err
	}
	group, err := d.Perm.GetGroup(path)
	if err != nil {
		return nil, err
	}
	return &FileInfo{f, mode, owner, group}, nil
}

func (d *Driver) ListDir(path string, callback func(ftp.FileInfo) error) error {
	basepath := d.realPath(path)
	return filepath.Walk(basepath, func(f string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rPath, _ := filepath.Rel(basepath, f)
		if rPath == info.Name() {
			mode, err := d.Perm.GetMode(rPath)
			if err != nil {
				return err
			}
			if info.IsDir() {
				mode |= os.ModeDir
			}
			owner, err := d.Perm.GetOwner(rPath)
			if err != nil {
				return err
			}
			group, err := d.Perm.GetGroup(rPath)
			if err != nil {
				return err
			}
			err = callback(&FileInfo{info, mode, owner, group})
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
}

func (d *Driver) DeleteDir(path string) error {
	rPath := d.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if f.IsDir() {
		return os.Remove(rPath)
	}
	return errors.New("Not a directory")
}

func (d *Driver) DeleteFile(path string) error {
	rPath := d.realPath(path)
	f, err := os.Lstat(rPath)
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return os.Remove(rPath)
	}
	return errors.New("Not a file")
}

func (d *Driver) Rename(fromPath string, toPath string) error {
	oldPath := d.realPath(fromPath)
	newPath := d.realPath(toPath)
	return os.Rename(oldPath, newPath)
}

func (d *Driver) MakeDir(path string) error {
	rPath := d.realPath(path)
	return os.MkdirAll(rPath, os.ModePerm)
}

func (d *Driver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	rPath := d.realPath(path)
	f, err := os.Open(rPath)
	if err != nil {
		return 0, nil, err
	}

	info, err := f.Stat()
	if err != nil {
		return 0, nil, err
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, nil, err
	}

	return info.Size(), f, nil
}

func (d *Driver) PutFile(destPath string, offset int64, data io.Reader, appendData bool) (int64, error) {
	rPath := d.realPath(destPath)
	var isExist bool
	f, err := os.Lstat(rPath)
	if err == nil {
		isExist = true
		if f.IsDir() {
			return 0, errors.New("A dir has the same name")
		}
	} else {
		if os.IsNotExist(err) {
			isExist = false
		} else {
			return 0, errors.New(fmt.Sprintln("Put File error:", err))
		}
	}

	if appendData && !isExist {
		appendData = false
	}

	if !appendData {
		if isExist {
			err = os.Remove(rPath)
			if err != nil {
				return 0, err
			}
		}
		f, err := os.Create(rPath)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		if _, err = f.Seek(offset, io.SeekStart); err != nil {
			return 0, err
		}
		n, err := io.Copy(f, data)
		if err != nil {
			return 0, err
		}
		return n, f.Close()
	}

	flags := os.O_WRONLY
	if offset == 0 {
		flags |= os.O_APPEND
	}

	of, err := os.OpenFile(rPath, flags, 0660)
	if err != nil {
		return 0, err
	}
	defer of.Close()

	if offset == 0 {
		_, err = of.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, err
		}
	} else {
		if _, err := of.Seek(offset, io.SeekStart); err != nil {
			return 0, err
		}
	}

	n, err := io.Copy(of, data)
	if err != nil {
		return 0, err
	}
	if err = of.Close(); err != nil {
		return 0, err
	}
	return n, nil
}

type Factory struct {
	ftp.Perm

	RootPath string
}

func (d *Driver) Abort() error { return nil } // noop

func (f *Factory) NewDriver() (ftp.Driver, error) {
	return &Driver{f.RootPath, f.Perm}, nil
}
