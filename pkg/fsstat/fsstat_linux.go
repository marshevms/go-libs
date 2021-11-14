//go:build linux
// +build linux

package fsstat

// #include <stdio.h>
// #include <mntent.h>
// #include "utils.h"
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

type MountEntry struct {
	Name    string   // Device or server for filesystem
	Dir     string   // Directory mounted on
	Type    string   // Type of filesystem: ufs, nfs, etc.
	Options []string // Comma-separated options for fs

	DumpFrequency int // Dump frequency (in days)
	PassNumber    int // Pass number for `fsck'
}

type Option struct {
	Name  string
	Value string
}

// MountInfo more info https://www.kernel.org/doc/Documentation/filesystems/proc.txt
type MountInfo struct {
	MountEntry

	MountID        int      // Unique identifier of the mount (may be reused after umount)
	ParentID       int      // ID of parent (or of self for the top of the mount tree)
	Major          int      // value of st_dev for files on filesystem
	Minor          int      // value of st_dev for files on filesystem
	Root           string   // root of the mount within the filesystem
	OptionalFields string   // zero or more fields of the form "tag[:value]"
	SuperOptions   []Option // per super block options
}

type mountInfoParser struct {
	file *os.File
	fp   *C.FILE

	reader *bufio.Reader
}

func newMountInfoParser() (*mountInfoParser, error) {
	parser := &mountInfoParser{}

	// try to read /proc/self/mountinfo
	parser.file, _ = os.OpenFile("/proc/self/mountinfo", os.O_RDONLY|syscall.O_CLOEXEC, 0)

	if parser.file == nil {
		mode := []byte{'r', 0}
		parser.file, _ = os.Open(C._PATH_MOUNTED)
		parser.fp = C.fdopen(C.int(parser.file.Fd()), (*C.char)(unsafe.Pointer(&mode[0])))
		if parser.fp == nil {
			return nil, fmt.Errorf("failed to open %s: %s", C._PATH_MOUNTED, C.GoString(C.getCError()))
		}

	} else {
		parser.reader = bufio.NewReaderSize(parser.file, 1024)
	}

	return parser, nil
}

func (p *mountInfoParser) close() error {
	return p.file.Close()
}

func (p *mountInfoParser) next() *MountInfo {
	mountInfo := &MountInfo{}

	if p.fp != nil {
		mnt := C.getmntent(p.fp)
		if mnt == nil {
			return nil
		}

		mountInfo.Name = C.GoString(mnt.mnt_fsname)
		mountInfo.Dir = C.GoString(mnt.mnt_dir)
		mountInfo.Type = C.GoString(mnt.mnt_type)
		mountInfo.Options = strings.Split(C.GoString(mnt.mnt_opts), ",")
		mountInfo.DumpFrequency = int(mnt.mnt_freq)
		mountInfo.PassNumber = int(mnt.mnt_passno)

		return mountInfo
	}

	line, err := p.reader.ReadString('\n')
	if err != nil {
		return nil
	}

	words := strings.Split(line, " ")
	if len(words) < 11 {
		return nil
	}

	mountInfo.Root = words[3]
	mountInfo.Dir = words[4]
	mountInfo.Options = strings.Split(words[5], ",")
	mountInfo.OptionalFields = words[6]
	mountInfo.Type = words[8]
	mountInfo.Name = words[9]

	superOptions := strings.Split(strings.TrimRight(words[10], "\n"), ",")
	mountInfo.SuperOptions = make([]Option, len(superOptions))

	for i, option := range superOptions {
		pair := strings.Split(option, "=")
		mountInfo.SuperOptions[i].Name = pair[0]

		if len(pair) == 2 {
			mountInfo.SuperOptions[i].Value = pair[1]
		}
	}

	if parseInt(&mountInfo.MountID, words[0]); err != nil {
		return nil
	}
	if parseInt(&mountInfo.ParentID, words[1]); err != nil {
		return nil
	}

	majorMinor := strings.Split(words[2], ":")
	if len(majorMinor) < 2 {
		return nil
	}

	if parseInt(&mountInfo.Major, majorMinor[0]); err != nil {
		return nil
	}
	if parseInt(&mountInfo.Minor, majorMinor[1]); err != nil {
		return nil
	}

	return mountInfo
}

func (fs *FSstat) initMountInfo(path string) error {
	var dirPathLength int

	parser, err := newMountInfoParser()
	if err != nil {
		return err
	}
	defer parser.close()

	for mountinfo := parser.next(); mountinfo != nil; mountinfo = parser.next() {

		if strings.HasPrefix(path, mountinfo.Dir) && dirPathLength < len(mountinfo.Dir) {
			dirPathLength = len(mountinfo.Dir)
			fs.mountInfo = mountinfo
		}
	}

	return nil
}

func (fs *FSstat) setPath(path string) error {
	var buf syscall.Statfs_t

	if err := syscall.Statfs(path, &buf); err != nil {
		return err
	}

	fs.capacity = buf.Blocks * uint64(buf.Frsize)
	fs.free = buf.Bfree * uint64(buf.Frsize)
	fs.available = buf.Bavail * uint64(buf.Frsize)

	fs.fstype = buf.Type
	fs.flags = buf.Flags

	return nil
}

func (fs *FSstat) Type() int64 {
	return fs.fstype
}

func MountInfoList() ([]*MountInfo, error) {
	parser, err := newMountInfoParser()
	if err != nil {
		return nil, err
	}
	defer parser.close()

	res := make([]*MountInfo, 0, 1)

	for mountinfo := parser.next(); mountinfo != nil; mountinfo = parser.next() {
		res = append(res, mountinfo)
	}

	return res, nil
}
