//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	shcore   = windows.NewLazySystemDLL("shcore.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pRegisterClassExW      = user32.NewProc("RegisterClassExW")
	pCreateWindowExW       = user32.NewProc("CreateWindowExW")
	pDefWindowProcW        = user32.NewProc("DefWindowProcW")
	pDestroyWindow         = user32.NewProc("DestroyWindow")
	pShowWindow            = user32.NewProc("ShowWindow")
	pUpdateWindow          = user32.NewProc("UpdateWindow")
	pGetMessageW           = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessageW      = user32.NewProc("DispatchMessageW")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pPostMessageW          = user32.NewProc("PostMessageW")
	pReleaseCapture        = user32.NewProc("ReleaseCapture")
	pGetCursorPos          = user32.NewProc("GetCursorPos")
	pGetWindowRect         = user32.NewProc("GetWindowRect")
	pSetWindowPos          = user32.NewProc("SetWindowPos")
	pSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	pGetDpiForSystem       = user32.NewProc("GetDpiForSystem")
	pGetDpiForWindow       = user32.NewProc("GetDpiForWindow")
	pMonitorFromRect       = user32.NewProc("MonitorFromRect")
	pFindWindowW           = user32.NewProc("FindWindowW")
	pSetForegroundWindow   = user32.NewProc("SetForegroundWindow")
	pIsIconic              = user32.NewProc("IsIconic")
	pMessageBoxW           = user32.NewProc("MessageBoxW")
	pLoadImageW            = user32.NewProc("LoadImageW")
	pLoadCursorW           = user32.NewProc("LoadCursorW")
	pGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	pCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	pShellExecuteW         = shell32.NewProc("ShellExecuteW")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pGetDpiForMonitor      = shcore.NewProc("GetDpiForMonitor")
	pGetModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
	pRtlMoveMemory         = kernel32.NewProc("RtlMoveMemory")

	pSetProcessDpiAwarenessContext       = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetThreadDpiAwarenessContext        = user32.NewProc("GetThreadDpiAwarenessContext")
	pGetAwarenessFromDpiAwarenessContext = user32.NewProc("GetAwarenessFromDpiAwarenessContext")
	pAreDpiAwarenessContextsEqual        = user32.NewProc("AreDpiAwarenessContextsEqual")
	pMonitorFromWindow                   = user32.NewProc("MonitorFromWindow")
	pMonitorFromPoint                    = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfoW                     = user32.NewProc("GetMonitorInfoW")
	pGetClientRect                       = user32.NewProc("GetClientRect")
)

const (
	wsPopup       = 0x80000000
	wsVisible     = 0x10000000
	wsClipChild   = 0x02000000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000

	wmDestroy       = 0x0002
	wmMove          = 0x0003
	wmSize          = 0x0005
	wmActivate      = 0x0006
	wmClose         = 0x0010
	wmNCLButtonDown = 0x00A1
	wmExitSizeMove  = 0x0232
	wmDpiChanged    = 0x02E0

	htCaption = 2

	swShowNormal = 1
	swMinimize   = 6
	swRestore    = 9

	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpFrameChanged = 0x0020

	spiGetWorkArea    = 0x0030
	monDefaultNull    = 0
	monDefaultPrimary = 1
	monDefaultNearest = 2

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 == (DPI_AWARENESS_CONTEXT)-4
	dpiCtxPerMonitorV2 = ^uintptr(3)

	mbOK          = 0x0
	mbYesNo       = 0x4
	mbIconError   = 0x10
	mbIconWarning = 0x30
	idYes         = 6

	imageIcon       = 1
	lrDefaultSize   = 0x40
	lrShared        = 0x8000
	idcArrow        = 32512
	smCxIcon        = 11
	smCyIcon        = 12
	smCxSmIcon      = 49
	smCySmIcon      = 50
	dwmwaCornerPref = 33
	dwmwcpRound     = 2
)

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type rect struct{ Left, Top, Right, Bottom int32 }
type point struct{ X, Y int32 }

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type msg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

func utf16(s string) *uint16 { p, _ := windows.UTF16PtrFromString(s); return p }

func messageBox(hwnd uintptr, text, caption string, flags uintptr) int {
	r, _, _ := pMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(utf16(text))), uintptr(unsafe.Pointer(utf16(caption))), flags)
	return int(r)
}

func shellOpen(target string) {
	pShellExecuteW.Call(0, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16(target))), 0, 0, swShowNormal)
}

func getWindowRect(hwnd uintptr) (rect, bool) {
	var r rect
	ok, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r, ok != 0
}

func workArea() rect {
	var r rect
	ok, _, _ := pSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	if ok == 0 || r.Right <= r.Left {
		r = rect{0, 0, 1920, 1040}
	}
	return r
}

func systemDPI() uint32 {
	if pGetDpiForSystem.Find() == nil {
		if d, _, _ := pGetDpiForSystem.Call(); d != 0 {
			return uint32(d)
		}
	}
	return 96
}

func windowDPI(hwnd uintptr) uint32 {
	if pGetDpiForWindow.Find() == nil {
		if d, _, _ := pGetDpiForWindow.Call(hwnd); d != 0 {
			return uint32(d)
		}
	}
	return systemDPI()
}

// monitorFor returns the monitor handle containing r (0 if r is off-screen).
func monitorFor(r rect) uintptr {
	h, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), monDefaultNull)
	return h
}

func monitorDPI(hmon uintptr) uint32 {
	if hmon != 0 && pGetDpiForMonitor.Find() == nil {
		var dx, dy uint32
		hr, _, _ := pGetDpiForMonitor.Call(hmon, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy)))
		if hr == 0 && dx != 0 {
			return dx
		}
	}
	return systemDPI()
}

func scale(v int32, dpi uint32) int32 { return int32((int64(v)*int64(dpi) + 48) / 96) }

// enableDPIAwareness is a belt-and-braces fallback for the manifest setting.
// It fails harmlessly (ERROR_ACCESS_DENIED) when the manifest already applied it.
func enableDPIAwareness() {
	if pSetProcessDpiAwarenessContext.Find() == nil {
		pSetProcessDpiAwarenessContext.Call(dpiCtxPerMonitorV2)
	}
}

// dpiAwarenessName reports the effective awareness of the UI thread.
func dpiAwarenessName() string {
	if pGetThreadDpiAwarenessContext.Find() != nil || pGetAwarenessFromDpiAwarenessContext.Find() != nil {
		return "unknown"
	}
	ctx, _, _ := pGetThreadDpiAwarenessContext.Call()
	a, _, _ := pGetAwarenessFromDpiAwarenessContext.Call(ctx)
	if pAreDpiAwarenessContextsEqual.Find() == nil {
		if eq, _, _ := pAreDpiAwarenessContextsEqual.Call(ctx, dpiCtxPerMonitorV2); eq != 0 {
			return "per-monitor-v2"
		}
	}
	switch int32(a) {
	case 0:
		return "UNAWARE (bitmap-stretched!)"
	case 1:
		return "system"
	case 2:
		return "per-monitor"
	}
	return "unknown"
}

// primaryMonitor returns the handle of the primary monitor.
func primaryMonitor() uintptr {
	h, _, _ := pMonitorFromPoint.Call(0, monDefaultPrimary) // POINT{0,0} passed by value
	return h
}

func monitorWorkArea(hmon uintptr) (rect, bool) {
	mi := monitorInfo{}
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	ok, _, _ := pGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi)))
	return mi.rcWork, ok != 0
}

func windowMonitor(hwnd uintptr) uintptr {
	h, _, _ := pMonitorFromWindow.Call(hwnd, monDefaultNearest)
	return h
}

func clientSize(hwnd uintptr) (int32, int32) {
	var r rect
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r.Right - r.Left, r.Bottom - r.Top
}

// readRect copies a RECT from a raw Win32 pointer (e.g. WM_DPICHANGED lParam)
// without converting a uintptr to a Go pointer.
func readRect(ptr uintptr) rect {
	var r rect
	pRtlMoveMemory.Call(uintptr(unsafe.Pointer(&r)), ptr, unsafe.Sizeof(r))
	return r
}
