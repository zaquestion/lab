// This file contains Darwin (MacOS) and *BSD specific calls.

//go:build freebsd || openbsd || dragonfly || darwin
// +build freebsd openbsd dragonfly darwin

package cmd

// Unfortunatelly MacOS don't have the DUP3() system call, which is forced
// by Linux ARM64 not having the DUP2() anymore. With that, we need to
// repeat the other code and func declarations that are the same.

// FIXME: there MUST be some better way to do that... only dupFD2() should
// be here.

import "syscall"

var (
	sysStdout = syscall.Stdout
	sysStderr = syscall.Stderr
)

func closeFD(fd int) error {
	return syscall.Close(fd)
}

func dupFD(fd int) (int, error) {
	return syscall.Dup(fd)
}

// From what I've seen, darwin is the only OS without DUP3() support
func dupFD2(newFD, oldFD int) error {
	return syscall.Dup2(newFD, oldFD)
}
