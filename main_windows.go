//go:build windows

// SingaporePSI.exe - a small frameless desktop widget showing Singapore PSI readings.
// Hosts the embedded HTML/JS UI in the WebView2 runtime that ships with Windows 11.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/edge"
	"github.com/wailsapp/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

//go:embed web
var webFS embed.FS

const (
	appVersion   = "1.2.0"
	appTitle     = "Singapore PSI"
	appName      = "Singapore PSI Widget"
	className    = "SingaporePSIWidgetWindow"
	mutexName    = "Local\\SingaporePSIWidget-Instance"
	virtualHost  = "psi-widget.example" // fixed origin => localStorage cache survives restarts
	startURL     = "https://" + virtualHost + "/index.html"
	baseW, baseH = 400, 540 // window size in logical px (DIPs) at 100% zoom (v1.2: +80 for timeline)
	margin       = 16
	webview2URL  = "https://developer.microsoft.com/microsoft-edge/webview2/"
)

// User text-size steps (ZoomFactor on top of the monitor's scale).
var zoomSteps = []float64{0.9, 1.0, 1.1, 1.25, 1.5, 1.75, 2.0}

var (
	mainHwnd    uintptr
	chromium    *edge.Chromium
	settingsDir string // %APPDATA%\PSIWidget
	dataDir     string // %LOCALAPPDATA%\PSIWidget
	debugMode   bool
	zoom        = 1.0
	runtimeVer  string
	rasterNote  string // how RasterizationScale was applied (for --debug)
	currentDPI  uint32 = 96
)

func init() { runtime.LockOSThread() }

type settings struct {
	X     *int32  `json:"x,omitempty"`
	Y     *int32  `json:"y,omitempty"`
	Zoom  float64 `json:"zoom,omitempty"`
	Debug bool    `json:"debug,omitempty"`
}

func loadSettings() settings {
	var s settings
	if b, err := os.ReadFile(filepath.Join(settingsDir, "settings.json")); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}

func saveSettings(s settings) {
	_ = os.MkdirAll(settingsDir, 0o755)
	b, _ := json.MarshalIndent(s, "", "  ")
	_ = os.WriteFile(filepath.Join(settingsDir, "settings.json"), b, 0o644)
}

func savePosition() {
	if mainHwnd == 0 {
		return
	}
	if ic, _, _ := pIsIconic.Call(mainHwnd); ic != 0 {
		return
	}
	r, ok := getWindowRect(mainHwnd)
	if !ok {
		return
	}
	s := loadSettings()
	s.X, s.Y = &r.Left, &r.Top
	s.Zoom = zoom
	saveSettings(s)
}

func clampZoom(z float64) float64 {
	if z < zoomSteps[0] || math.IsNaN(z) {
		if z == 0 || math.IsNaN(z) {
			return 1.0
		}
		return zoomSteps[0]
	}
	if z > zoomSteps[len(zoomSteps)-1] {
		return zoomSteps[len(zoomSteps)-1]
	}
	return z
}

// windowSize: logical 400x460 * (monitor DPI / 96) * user zoom, in physical pixels.
func windowSize(dpi uint32, z float64) (int32, int32) {
	return int32(math.Round(float64(baseW) * z * float64(dpi) / 96)),
		int32(math.Round(float64(baseH) * z * float64(dpi) / 96))
}

func fatal(text string) {
	messageBox(mainHwnd, text, appName, mbOK|mbIconError)
	os.Exit(1)
}

// extractAssets writes the embedded web UI to dir (only files whose content changed).
func extractAssets(dir string) error {
	return fs.WalkDir(webFS, "web", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, "web"), "/")
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := webFS.ReadFile(p)
		if err != nil {
			return err
		}
		if old, err := os.ReadFile(dst); err == nil && bytes.Equal(old, data) {
			return nil
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

func hasArg(name string) bool {
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, name) || strings.EqualFold(a, "/"+strings.TrimLeft(name, "-")) {
			return true
		}
	}
	return false
}

func main() {
	// Must happen before any window is created (the manifest normally does this already).
	enableDPIAwareness()

	// ---- single instance ----
	mh, err := windows.CreateMutex(nil, false, utf16(mutexName))
	if err == windows.ERROR_ALREADY_EXISTS {
		if h, _, _ := pFindWindowW.Call(uintptr(unsafe.Pointer(utf16(className))), 0); h != 0 {
			if ic, _, _ := pIsIconic.Call(h); ic != 0 {
				pShowWindow.Call(h, swRestore)
			}
			pSetForegroundWindow.Call(h)
		}
		return
	}
	defer windows.CloseHandle(mh)

	// ---- WebView2 runtime present? ----
	ver, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if err != nil || ver == "" {
		text := "This widget needs the Microsoft Edge WebView2 Runtime, which was not found on this PC.\n\n" +
			"WebView2 is normally built into Windows 11 (even if the Edge browser is removed). " +
			"You can install it for free (\"Evergreen Bootstrapper\") from:\n\n" + webview2URL +
			"\n\nOpen the download page now?"
		if err != nil {
			text += "\n\n(Details: " + err.Error() + ")"
		}
		if messageBox(0, text, appName, mbYesNo|mbIconWarning) == idYes {
			shellOpen(webview2URL)
		}
		os.Exit(2)
	}
	runtimeVer = ver

	// ---- folders ----
	local := os.Getenv("LOCALAPPDATA")
	roaming := os.Getenv("APPDATA")
	if local == "" {
		local = os.TempDir()
	}
	if roaming == "" {
		roaming = local
	}
	dataDir = filepath.Join(local, "PSIWidget")
	settingsDir = filepath.Join(roaming, "PSIWidget")
	appDir := filepath.Join(dataDir, "app")
	if err := extractAssets(appDir); err != nil {
		fatal("Could not write widget files to " + appDir + ":\n" + err.Error())
	}
	syncAutostartPath()
	cfg := loadSettings()
	zoom = clampZoom(cfg.Zoom)
	debugMode = hasArg("--debug") || cfg.Debug

	// ---- window ----
	hInst, _, _ := pGetModuleHandleW.Call(0)
	icoBig, _, _ := pLoadImageW.Call(hInst, uintptr(unsafe.Pointer(utf16("APP"))), imageIcon, 0, 0, lrDefaultSize|lrShared)
	cxs, _, _ := pGetSystemMetrics.Call(smCxSmIcon)
	cys, _, _ := pGetSystemMetrics.Call(smCySmIcon)
	icoSm, _, _ := pLoadImageW.Call(hInst, uintptr(unsafe.Pointer(utf16("APP"))), imageIcon, cxs, cys, lrShared)
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	brush, _, _ := pCreateSolidBrush.Call(0x001E1612) // RGB(18,22,30) as 0x00BBGGRR
	wc := wndClassEx{
		lpfnWndProc:   windows.NewCallback(wndProc),
		hInstance:     hInst,
		hIcon:         icoBig,
		hIconSm:       icoSm,
		hCursor:       cursor,
		hbrBackground: brush,
		lpszClassName: utf16(className),
	}
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	if r, _, e := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		fatal("RegisterClassEx failed: " + e.Error())
	}

	x, y, w, h := initialPlacement(cfg)
	mainHwnd, _, err = pCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(utf16(className))), uintptr(unsafe.Pointer(utf16(appTitle))),
		wsPopup|wsClipChild|wsSysMenu|wsMinimizeBox,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, hInst, 0)
	if mainHwnd == 0 {
		fatal("CreateWindowEx failed: " + err.Error())
	}
	// Re-sync the size with the DPI Windows actually assigned to the window
	// (covers stale "system DPI" after a scale change without signing out).
	currentDPI = windowDPI(mainHwnd)
	zoom = math.Min(zoom, maxZoom())
	fitWindow(false)

	// Windows 11 rounded corners (ignored on Windows 10)
	pref := uint32(dwmwcpRound)
	pDwmSetWindowAttribute.Call(mainHwnd, dwmwaCornerPref, uintptr(unsafe.Pointer(&pref)), 4)

	// ---- WebView2 ----
	chromium = edge.NewChromium()
	chromium.DataPath = filepath.Join(dataDir, "WebView2")
	chromium.Debug = debugMode
	chromium.SetErrorCallback(func(err error) {
		fatal("WebView2 error: " + err.Error() +
			"\n\nIf this keeps happening, (re)install the WebView2 Runtime from\n" + webview2URL)
	})
	chromium.MessageCallback = func(m string, _ *edge.ICoreWebView2, _ *edge.ICoreWebView2WebMessageReceivedEventArgs) {
		handleMessage(m)
	}
	pShowWindow.Call(mainHwnd, swShowNormal)
	pUpdateWindow.Call(mainHwnd)

	if !chromium.Embed(mainHwnd) {
		fatal("Could not start the WebView2 browser control.")
	}
	chromium.SetBackgroundColour(18, 22, 30, 255)
	applyContentScale()
	chromium.Resize() // controller bounds = client rect in physical px (library uses RAW_PIXELS bounds mode)
	if s, err := chromium.GetSettings(); err == nil {
		_ = s.PutAreDefaultContextMenusEnabled(debugMode)
		_ = s.PutAreDevToolsEnabled(debugMode)
		_ = s.PutIsZoomControlEnabled(false) // Ctrl+wheel / Ctrl+± are handled by the page -> host zoom
		_ = s.PutIsStatusBarEnabled(false)
	}
	wv3 := chromium.GetICoreWebView2_3()
	if wv3 == nil {
		fatal("Your WebView2 Runtime is too old. Please update it from\n" + webview2URL)
	}
	if err := wv3.SetVirtualHostNameToFolderMapping(virtualHost, appDir, edge.COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_DENY_CORS); err != nil {
		fatal("Could not map widget files: " + err.Error())
	}
	chromium.Navigate(startURL)

	// ---- message loop ----
	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// initialPlacement returns the window rectangle in physical pixels: the saved
// position if it is still on a connected monitor, otherwise top-right of the
// primary monitor's work area. Size uses the DPI of the target monitor.
func initialPlacement(cfg settings) (x, y, w, h int32) {
	z := clampZoom(cfg.Zoom)
	if cfg.X != nil && cfg.Y != nil {
		probe := rect{*cfg.X, *cfg.Y, *cfg.X + 40, *cfg.Y + 40}
		if mon := monitorFor(probe); mon != 0 {
			w, h = windowSize(monitorDPI(mon), z)
			return *cfg.X, *cfg.Y, w, h
		}
	}
	mon := primaryMonitor()
	dpi := monitorDPI(mon)
	wa, ok := monitorWorkArea(mon)
	if !ok {
		wa = workArea()
	}
	w, h = windowSize(dpi, math.Min(z, fitZoom(dpi, wa)))
	m := scale(margin, dpi)
	return wa.Right - w - m, wa.Top + m, w, h
}

// fitWindow sets the window size for currentDPI and zoom. With anchorRight the
// right edge stays put (the widget normally lives at the top-right). The window
// is kept inside its monitor's work area.
func fitWindow(anchorRight bool) {
	r, ok := getWindowRect(mainHwnd)
	if !ok {
		return
	}
	w, h := windowSize(currentDPI, zoom)
	x, y := r.Left, r.Top
	if anchorRight {
		x = r.Right - w
	}
	if wa, ok := monitorWorkArea(windowMonitor(mainHwnd)); ok {
		if x+w > wa.Right {
			x = wa.Right - w
		}
		if y+h > wa.Bottom {
			y = wa.Bottom - h
		}
		if x < wa.Left {
			x = wa.Left
		}
		if y < wa.Top {
			y = wa.Top
		}
	}
	pSetWindowPos.Call(mainHwnd, 0, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoZOrder|swpNoActivate)
}

// putRasterizationScale calls ICoreWebView2Controller3::put_RasterizationScale(double).
// NOTE: go-webview2 v1.0.23's own PutRasterizationScale passes a *pointer* to the
// double instead of the value, so WebView2 receives garbage (~1e-310) and the
// call fails/misbehaves. On Windows x64 the Go syscall trampoline mirrors integer
// argument registers into XMM0-3, so passing the IEEE-754 bits works for a double.
func putRasterizationScale(c3 *edge.ICoreWebView2Controller3, s float64) error {
	hr, _, _ := c3.Vtbl.PutRasterizationScale.Call(uintptr(unsafe.Pointer(c3)), uintptr(math.Float64bits(s)))
	if hr != 0 {
		return windows.Errno(hr)
	}
	return nil
}

// applyContentScale renders the page at the monitor scale (dpi/96) and applies the user zoom.
func applyContentScale() {
	if chromium == nil || chromium.GetController() == nil {
		return
	}
	want := float64(currentDPI) / 96.0
	rasterNote = "n/a"
	if c3 := chromium.GetController().GetICoreWebView2Controller3(); c3 != nil {
		_ = c3.PutShouldDetectMonitorScaleChanges(false) // we drive it from WM_DPICHANGED
		if err := putRasterizationScale(c3, want); err != nil {
			// Fallback: let WebView2 follow the monitor itself.
			_ = c3.PutShouldDetectMonitorScaleChanges(true)
			rasterNote = "auto (" + err.Error() + ")"
		} else {
			rasterNote = "set"
		}
	}
	_ = chromium.GetController().PutZoomFactor(zoom)
}

// maxZoom is the largest zoom step whose window still fits the monitor's work area.
// If even the smallest step does not fit (very small screens / high scale),
// it returns the exact factor that fits (not below 0.6).
func maxZoom() float64 {
	wa, ok := monitorWorkArea(windowMonitor(mainHwnd))
	if !ok {
		return zoomSteps[len(zoomSteps)-1]
	}
	return fitZoom(currentDPI, wa)
}

func fitZoom(dpi uint32, wa rect) float64 {
	best := 0.0
	for _, z := range zoomSteps {
		w, h := windowSize(dpi, z)
		if w <= wa.Right-wa.Left && h <= wa.Bottom-wa.Top {
			best = z
		}
	}
	if best == 0 {
		fw := float64(wa.Right-wa.Left) / (float64(baseW) * float64(dpi) / 96)
		fh := float64(wa.Bottom-wa.Top) / (float64(baseH) * float64(dpi) / 96)
		best = math.Max(0.6, math.Floor(math.Min(fw, fh)*100)/100)
	}
	return best
}

func setZoom(z float64) {
	zoom = math.Min(clampZoom(z), maxZoom())
	applyContentScale()
	fitWindow(true)
	chromium.Resize()
	savePosition()
	replyState()
}

func stepZoom(dir int) {
	// find nearest step
	idx := 0
	for i, s := range zoomSteps {
		if math.Abs(s-zoom) < math.Abs(zoomSteps[idx]-zoom) {
			idx = i
		}
	}
	idx += dir
	if idx < 0 {
		idx = 0
	}
	if idx >= len(zoomSteps) {
		idx = len(zoomSteps) - 1
	}
	setZoom(zoomSteps[idx])
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmSize:
		if chromium != nil && hwnd == mainHwnd {
			chromium.Resize()
		}
		return 0
	case wmMove:
		if chromium != nil && hwnd == mainHwnd {
			_ = chromium.NotifyParentWindowPositionChanged()
		}
		return 0
	case wmExitSizeMove:
		savePosition()
		return 0
	case wmDpiChanged:
		// Per-monitor v2: move/resize to the rect Windows suggests, then make the
		// WebView2 follow: bounds (physical px) + RasterizationScale = dpi/96.
		currentDPI = uint32(wParam & 0xFFFF)
		r := readRect(lParam)
		pSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top),
			uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), swpNoZOrder|swpNoActivate)
		if hwnd == mainHwnd {
			fitWindow(false) // exact size for new DPI (guards against rounding drift)
			applyContentScale()
			if chromium != nil {
				chromium.Resize()
				replyState()
			}
		}
		return 0
	case wmClose:
		savePosition()
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if chromium != nil {
			chromium.ShuttingDown()
		}
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

type hostMsg struct {
	Cmd string          `json:"cmd"`
	Arg json.RawMessage `json:"arg"`
}

// handleMessage runs on the UI thread (WebView2 callbacks are delivered there).
func handleMessage(raw string) {
	var m hostMsg
	if json.Unmarshal([]byte(raw), &m) != nil {
		return
	}
	switch m.Cmd {
	case "drag":
		var pt point
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		pReleaseCapture.Call()
		lp := uintptr(uint32(uint16(pt.X)) | uint32(uint16(pt.Y))<<16)
		pPostMessageW.Call(mainHwnd, wmNCLButtonDown, htCaption, lp)
	case "close":
		pPostMessageW.Call(mainHwnd, wmClose, 0, 0)
	case "minimize":
		pShowWindow.Call(mainHwnd, swMinimize)
	case "hello", "getAutostart":
		_ = chromium.GetController().PutZoomFactor(zoom)
		replyAutostart(nil)
		replyState()
	case "autostart":
		var on bool
		_ = json.Unmarshal(m.Arg, &on)
		replyAutostart(setAutostart(on))
	case "zoom":
		var step int
		_ = json.Unmarshal(m.Arg, &step)
		if step == 0 {
			setZoom(1.0)
		} else if step > 0 {
			stepZoom(1)
		} else {
			stepZoom(-1)
		}
	case "open":
		var u string
		if json.Unmarshal(m.Arg, &u) == nil && strings.HasPrefix(strings.ToLower(u), "https://") && !strings.ContainsAny(u, "\"\r\n") {
			shellOpen(u)
		}
	}
}

func replyAutostart(setErr error) {
	errJS := "null"
	if setErr != nil {
		b, _ := json.Marshal(setErr.Error())
		errJS = string(b)
	}
	chromium.Eval(fmt.Sprintf("window.psiHost && window.psiHost.autostart(%t, %s)", autostartEnabled(), errJS))
}

// replyState tells the page the current zoom (and diagnostics in --debug mode).
func replyState() {
	dbg := ""
	if debugMode {
		raster := -1.0
		if c3 := chromium.GetController().GetICoreWebView2Controller3(); c3 != nil {
			raster, _ = c3.GetRasterizationScale()
		}
		zf, _ := chromium.GetController().GetZoomFactor()
		cw, ch := clientSize(mainHwnd)
		dbg = fmt.Sprintf("v%s | DPI %d (%.0f%%) %s | raster %.2f [%s] | zoom %.2f | client %dx%d px | WebView2 %s",
			appVersion, windowDPI(mainHwnd), float64(windowDPI(mainHwnd))/0.96, dpiAwarenessName(),
			raster, rasterNote, zf, cw, ch, runtimeVer)
	}
	b, _ := json.Marshal(dbg)
	chromium.Eval(fmt.Sprintf("window.psiHost && window.psiHost.state(%v, %s, %v, %v)",
		zoom, string(b), zoom <= zoomSteps[0]+1e-9, zoom >= maxZoom()-1e-9))
}
