//go:build !windows

package main

import (
	"fmt"
	"os"
)

func pause(msg string) {
	fmt.Println(msg)
	fmt.Println("Press Enter to exit...")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
}
