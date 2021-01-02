// +build !windows

package main

import "syscall"

func getCPU() int64 {
	usage := new(syscall.Rusage)
	syscall.Getrusage(syscall.RUSAGE_SELF, usage)
	return usage.Utime.Nano() + usage.Stime.Nano()
}
