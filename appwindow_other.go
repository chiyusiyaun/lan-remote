//go:build !windows

package main

// Linux: no WebView2; caller falls back to browser.
func openAppWindow(title, url string, w, h int) bool { return false }
