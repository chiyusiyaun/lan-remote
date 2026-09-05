//go:build linux

package discovery

func setSockBroadcast(fd uintptr) error { return nil }
func setReuse(fd uintptr) error         { return nil }
