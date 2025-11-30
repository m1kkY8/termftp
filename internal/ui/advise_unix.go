//go:build unix

package ui

import (
	"os"

	"golang.org/x/sys/unix"
)

func adviseSequential(f *os.File) {
	if f == nil {
		return
	}
	fd := int(f.Fd())
	_ = unix.Fadvise(fd, 0, 0, unix.FADV_SEQUENTIAL)
}
