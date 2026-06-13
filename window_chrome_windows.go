package main

import (
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

var (
	dwmapi               = windows.NewLazySystemDLL("dwmapi.dll")
	procDwmSetWindowAttr = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	dwmwaUseImmersiveDarkMode       = 20
	dwmwaUseImmersiveDarkModeLegacy = 19
	dwmwaBorderColor                = 34
	dwmwaCaptionColor               = 35
	dwmwaTextColor                  = 36
)

func applyDarkWindowChrome(w webview2.WebView) {
	if w == nil || w.Window() == nil {
		return
	}
	hwnd := uintptr(w.Window())
	enabled := uint32(1)
	setDwmAttribute(hwnd, dwmwaUseImmersiveDarkMode, enabled)
	setDwmAttribute(hwnd, dwmwaUseImmersiveDarkModeLegacy, enabled)
	setDwmAttribute(hwnd, dwmwaCaptionColor, colorRef(0x0f, 0x18, 0x23))
	setDwmAttribute(hwnd, dwmwaBorderColor, colorRef(0x21, 0x34, 0x48))
	setDwmAttribute(hwnd, dwmwaTextColor, colorRef(0xef, 0xf5, 0xfb))
}

func setDwmAttribute(hwnd uintptr, attr uint32, value uint32) {
	if hwnd == 0 {
		return
	}
	_, _, _ = procDwmSetWindowAttr.Call(
		hwnd,
		uintptr(attr),
		uintptr(unsafe.Pointer(&value)),
		unsafe.Sizeof(value),
	)
}

func colorRef(r, g, b uint8) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}
