//go:build windows

package discovery

import (
	"syscall"
	"unsafe"
)

var (
	ws2           = syscall.NewLazyDLL("ws2_32.dll")
	procSetSocket = ws2.NewProc("setsockopt")
)

const (
	solSocket   = 0xffff
	soBroadcast = 0x20
	soReuseAddr = 4
)

func setSockBroadcast(fd uintptr) error {
	v := int32(1)
	_, _, err := procSetSocket.Call(fd, solSocket, soBroadcast, uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	if err != nil && err.(syscall.Errno) != 0 {
		return err
	}
	return nil
}

func setReuse(fd uintptr) error {
	v := int32(1)
	_, _, err := procSetSocket.Call(fd, solSocket, soReuseAddr, uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	if err != nil && err.(syscall.Errno) != 0 {
		return err
	}
	return nil
}
