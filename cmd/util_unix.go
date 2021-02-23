// This file contains Linux specific calls.

// +build !windows

package cmd

// Since we're using some system calls that are platform-specific, we need
// to make sure we have a small layer of compatibility for Unix-like and
// Windows operating systems. For now, this file is still valid for BSDs
// (MacOS included).

import "syscall"

// We're using the Linux API as primary model, hence we can only return
// the results from the default syscalls.

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

func dupFD2(newFD, oldFD int) error {
	return syscall.Dup2(newFD, oldFD)
}
