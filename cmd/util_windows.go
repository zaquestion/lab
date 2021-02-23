// This file contains Windows specific calls.

// +build windows

package cmd

// Even though Windows has a POSIX layer, it's implemented in userspace and,
// consequently, the "syscall" lib doesn't export it.
// Because of it, we need to use specific Windows calls to handle some of the
// syscall we're using the `lab`.

import "syscall"

// SetStdHandle is not exported by golang syscall lib, we need to get it
// ourselves from kernel32.dll.
var (
	kernel32             = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandleAddr = kernel32.MustFindProc("SetStdHandle").Addr()
)

// Windows has the concept of "Handles", which in Unix can be directly
// converted to integers.
var (
	sysStdout = int(syscall.Stdout)
	sysStderr = int(syscall.Stderr)
)

// closeFD behaves the as POSIX close()
func closeFD(fd int) error {
	return syscall.Close(syscall.Handle(fd))
}

// dupFD behaves the same as POSIX dup()
func dupFD(fd int) (int, error) {
	proc, err := syscall.GetCurrentProcess()
	if err != nil {
		return 0, err
	}

	var hndl syscall.Handle
	err = syscall.DuplicateHandle(proc, syscall.Handle(fd), proc, &hndl, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	return int(hndl), err
}

// dupFD2 behaves the same as POSIX dup2()
func dupFD2(oldFD, newFD int) error {
	ret, _, err := syscall.Syscall(procSetStdHandleAddr, 2, uintptr(oldFD), uintptr(newFD), 0)
	if err != 0 {
		return error(err)
	}

	if ret == 0 {
		return syscall.EINVAL
	}

	return nil
}
