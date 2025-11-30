//go:build !unix

package ui

import "os"

func adviseSequential(f *os.File) {}
