package fsstat

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"
)

func Test_fsstat(t *testing.T) {
	test, err := NewFSstat("/")

	fmt.Println(test.Type() == unix.EXT4_SUPER_MAGIC, err)

	res, _ := MountInfoList()

	for _, v := range res {
		fmt.Println(v.SuperOptions)
	}
}
