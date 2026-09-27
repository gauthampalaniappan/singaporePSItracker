//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"
)

// TestAutostartRoundTrip writes to HKCU\...\Run, so it only runs when explicitly
// requested (PSI_TEST_REGISTRY=1) and restores any existing entry afterwards.
func TestAutostartRoundTrip(t *testing.T) {
	if os.Getenv("PSI_TEST_REGISTRY") != "1" {
		t.Skip("set PSI_TEST_REGISTRY=1 to run (modifies HKCU Run key)")
	}
	wasOn := autostartEnabled()
	defer func() { _ = setAutostart(wasOn) }()
	if err := setAutostart(true); err != nil {
		t.Fatal(err)
	}
	if !autostartEnabled() {
		t.Fatal("expected enabled")
	}
	if err := setAutostart(false); err != nil {
		t.Fatal(err)
	}
	if autostartEnabled() {
		t.Fatal("expected disabled")
	}
	if err := setAutostart(false); err != nil { // idempotent
		t.Fatal(err)
	}
}

func TestExtractAssets(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := extractAssets(dir); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"index.html", "vendor/leaflet/leaflet.js", "vendor/leaflet/leaflet.css"} {
		if st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil || st.Size() == 0 {
			t.Fatalf("%s missing: %v", f, err)
		}
	}
}

func TestPlacementAndSettings(t *testing.T) {
	settingsDir = t.TempDir()
	x, y, w, h := initialPlacement(loadSettings())
	wa := workArea()
	t.Logf("work area %+v dpi %d -> window %d,%d %dx%d", wa, systemDPI(), x, y, w, h)
	if w <= 0 || h <= 0 || x+w > wa.Right || y < wa.Top {
		t.Fatal("bad default placement")
	}
	px, py := int32(100), int32(120)
	saveSettings(settings{X: &px, Y: &py})
	x2, y2, _, _ := initialPlacement(loadSettings())
	if x2 != 100 || y2 != 120 {
		t.Fatalf("saved position not used: %d,%d", x2, y2)
	}
	ox, oy := int32(-50000), int32(-50000)
	saveSettings(settings{X: &ox, Y: &oy})
	x3, _, _, _ := initialPlacement(loadSettings())
	if x3 == ox {
		t.Fatal("off-screen position should be ignored")
	}
}

func TestScaling(t *testing.T) {
	cases := []struct {
		dpi  uint32
		z    float64
		w, h int32
	}{{96, 1, 400, 540}, {144, 1, 600, 810}, {192, 1, 800, 1080}, {120, 1.25, 625, 844}, {144, 2, 1200, 1620}}
	for _, c := range cases {
		if w, h := windowSize(c.dpi, c.z); w != c.w || h != c.h {
			t.Errorf("windowSize(%d,%.2f)=%dx%d want %dx%d", c.dpi, c.z, w, h, c.w, c.h)
		}
	}
	if clampZoom(0) != 1 || clampZoom(5) != 2 || clampZoom(0.5) != 0.9 || clampZoom(1.25) != 1.25 {
		t.Error("clampZoom")
	}
	src := rect{1, 2, 3, 4}
	if got := readRect(uintptr(unsafe.Pointer(&src))); got != src {
		t.Errorf("readRect %+v", got)
	}
	// 1280x720 @150%: work area 672 px high -> no step fits, exact factor used
	if z := fitZoom(144, rect{0, 0, 1280, 672}); z < 0.6 || z > 0.83 {
		t.Errorf("fitZoom small screen = %.2f", z)
	}
	if z := fitZoom(144, rect{0, 0, 1920, 1032}); z != 1.25 {
		t.Errorf("fitZoom 1080p@150%% = %.2f want 1.25", z)
	}
	enableDPIAwareness()
	t.Logf("awareness=%s primaryDPI=%d", dpiAwarenessName(), monitorDPI(primaryMonitor()))
}
