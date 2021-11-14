package fsstat

import (
	"math"
	"syscall"
)

type FSstat struct {
	capacity  uint64
	free      uint64
	available uint64
	fstype    int64
	flags     int64

	mountInfo *MountInfo
	path      string
}

func NewFSstat(path string) (*FSstat, error) {
	fsstat := FSstat{}
	return fsstat.invalidate(), fsstat.update()
}

func (fs *FSstat) Update() error {
	return fs.update()
}

func (fs *FSstat) SetPath(path string) error {
	fs.invalidate()
	fs.path = path

	return fs.update()
}

func (fs *FSstat) BytesCapacity() uint64 {
	return fs.capacity
}

func (fs *FSstat) BytesFree() uint64 {
	return fs.free
}

func (fs *FSstat) BytesAvailable() uint64 {
	return fs.available
}

func (fs *FSstat) IsReadOnly() bool {
	return fs.flags&syscall.MS_RDONLY == 1
}

func (fs *FSstat) update() error {
	if err := fs.setPath(fs.path); err != nil {
		return err
	}
	if err := fs.initMountInfo(fs.path); err != nil {
		return err
	}

	return nil
}

func (fs *FSstat) invalidate() *FSstat {
	fs.capacity = math.MaxUint64
	fs.free = math.MaxUint64
	fs.available = math.MaxUint64

	fs.flags = 0
	fs.fstype = 0

	fs.mountInfo = nil

	fs.path = ""

	return fs
}
