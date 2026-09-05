//go:build windows

package main

import (
	"fmt"

	webview2 "github.com/jchv/go-webview2"
)

// openAppWindow shows a native desktop window (WebView2).
// Returns false ONLY if the webview could not be created.
func openAppWindow(title, url string, w, h int) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("webview panic:", r)
		}
	}()

	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  uint(w),
			Height: uint(h),
			Center: true,
		},
	})
	if wv == nil {
		fmt.Println("ERROR: WebView2 window creation returned nil")
		return false
	}
	defer wv.Destroy()
	wv.Navigate(url)
	fmt.Println("Native app window opened (not a browser).")
	wv.Run()
	return true
}
