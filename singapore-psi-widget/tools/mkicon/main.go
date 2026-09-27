// mkicon draws the 256x256 app icon (a PSI "gauge") as PNG. Run: go run ./tools/mkicon out.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type rgba struct{ r, g, b, a float64 }

func main() {
	const S, SS = 256, 4 // size, supersampling
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	segs := []rgba{{46, 204, 113, 1}, {52, 152, 219, 1}, {241, 196, 15, 1}, {230, 126, 34, 1}, {231, 76, 60, 1}}
	cx, cy := 128.0, 162.0
	needle := math.Pi * (1 - 0.42) // angle for needle (0=right, pi=left)
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			var acc rgba
			for sy := 0; sy < SS; sy++ {
				for sx := 0; sx < SS; sx++ {
					px := float64(x) + (float64(sx)+0.5)/SS
					py := float64(y) + (float64(sy)+0.5)/SS
					c := sample(px, py, cx, cy, needle, segs)
					acc.r += c.r * c.a
					acc.g += c.g * c.a
					acc.b += c.b * c.a
					acc.a += c.a
				}
			}
			n := float64(SS * SS)
			if acc.a > 0 {
				img.SetNRGBA(x, y, color.NRGBA{uint8(acc.r / acc.a), uint8(acc.g / acc.a), uint8(acc.b / acc.a), uint8(255 * acc.a / n)})
			}
		}
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func sample(px, py, cx, cy, needle float64, segs []rgba) rgba {
	// rounded square background
	const pad, rad = 8.0, 52.0
	qx := math.Max(math.Abs(px-128)-(128-pad-rad), 0)
	qy := math.Max(math.Abs(py-128)-(128-pad-rad), 0)
	if math.Hypot(qx, qy) > rad {
		return rgba{}
	}
	out := rgba{20, 26, 36, 1}
	dx, dy := px-cx, cy-py
	d := math.Hypot(dx, dy)
	ang := math.Atan2(dy, dx) // 0..pi on upper half
	// gauge arc
	if d >= 72 && d <= 102 && dy >= 0 {
		if ang < 0 {
			ang = 0
		}
		t := 1 - ang/math.Pi // 0 at left, 1 at right
		i := int(t * 5)
		if i > 4 {
			i = 4
		}
		// small gaps between segments
		frac := t*5 - float64(i)
		if frac > 0.04 && frac < 0.96 || (i == 0 && frac <= 0.04) || (i == 4 && frac >= 0.96) {
			return segs[i]
		}
	}
	// needle (thick line from centre)
	ux, uy := math.Cos(needle), math.Sin(needle)
	along := dx*ux + dy*uy
	perp := math.Abs(-dx*uy + dy*ux)
	if along >= 0 && along <= 88 && perp <= 7-along*0.05 {
		return rgba{240, 244, 248, 1}
	}
	if d <= 16 {
		return rgba{240, 244, 248, 1}
	}
	return out
}
