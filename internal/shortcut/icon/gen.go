//go:build ignore

// gen draws the roundhouse icon (a railway turntable) and writes it as roundhouse.png (256 px),
// roundhouse.ico and roundhouse.icns. The shapes are the ones of ../../../docs/logo.svg; keep both in step.
//
//	go run gen.go
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

var (
	slate = color.NRGBA{0x1f, 0x29, 0x37, 0xff}
	amber = color.NRGBA{0xf5, 0x9e, 0x0b, 0xff}
	white = color.NRGBA{0xf9, 0xfa, 0xfb, 0xff}
)

// shapes in a 256-unit square, painted in this order; the first that covers a point wins
// from the top, so the list reads top layer first.
var layers = []struct {
	c      color.NRGBA
	covers func(x, y float64) bool
}{
	{amber, func(x, y float64) bool { return math.Hypot(x-128, y-128) <= 12 }},                     // hub
	{white, func(x, y float64) bool { return segment(x, y, 128-44, 128+44, 128+44, 128-44) <= 9 }}, // bridge
	{amber, func(x, y float64) bool { return math.Abs(math.Hypot(x-128, y-128)-62) <= 7 }},         // ring
	{white, func(x, y float64) bool { return ray(x, y, 3, 78, 112) <= 8 }},                         // track that leads in
	{amber, stalls},
	{slate, func(x, y float64) bool { return roundedRect(x, y, 256, 56) }}, // background
}

// stalls are the five bays of the roundhouse: left, the upper half and right. The track that leads in
// is at the lower left; the bridge of the turntable lines up with it and with the bay opposite.
func stalls(x, y float64) bool {
	for _, i := range []int{0, 4, 5, 6, 7} {
		if ray(x, y, i, 76, 104) <= 8 {
			return true
		}
	}
	return false
}

// segment is the distance from (x, y) to the line from (x1, y1) to (x2, y2).
func segment(x, y, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	t := math.Max(0, math.Min(1, ((x-x1)*dx+(y-y1)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(x-(x1+t*dx), y-(y1+t*dy))
}

func roundedRect(x, y, size, r float64) bool {
	cx := math.Max(r, math.Min(size-r, x))
	cy := math.Max(r, math.Min(size-r, y))
	return math.Hypot(x-cx, y-cy) <= r
}

// render draws the icon at px pixels, with 4x4 samples per pixel for smooth edges.
func render(px int) *image.NRGBA {
	const n = 4
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	scale := 256 / float64(px)
	for py := 0; py < px; py++ {
		for pxl := 0; pxl < px; pxl++ {
			var r, g, b, a float64
			for sy := 0; sy < n; sy++ {
				for sx := 0; sx < n; sx++ {
					x := (float64(pxl) + (float64(sx)+0.5)/n) * scale
					y := (float64(py) + (float64(sy)+0.5)/n) * scale
					for _, l := range layers {
						if l.covers(x, y) {
							r, g, b, a = r+float64(l.c.R), g+float64(l.c.G), b+float64(l.c.B), a+255
							break
						}
					}
				}
			}
			if a > 0 {
				k := float64(n * n)
				img.SetNRGBA(pxl, py, color.NRGBA{uint8(r*255/a + 0.5), uint8(g*255/a + 0.5), uint8(b*255/a + 0.5), uint8(a/k + 0.5)})
			}
		}
	}
	return img
}

func encode(px int) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, render(px)); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

// ico packs PNG images (supported since Windows Vista) into one icon file.
func ico(sizes ...int) []byte {
	var head, data bytes.Buffer
	binary.Write(&head, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for _, s := range sizes {
		p := encode(s)
		dim := byte(s)
		if s >= 256 {
			dim = 0
		}
		head.Write([]byte{dim, dim, 0, 0})
		binary.Write(&head, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&head, binary.LittleEndian, [2]uint32{uint32(len(p)), uint32(offset)})
		data.Write(p)
		offset += len(p)
	}
	return append(head.Bytes(), data.Bytes()...)
}

// icns packs PNG images into an Apple icon file.
func icns() []byte {
	var body bytes.Buffer
	for _, e := range []struct {
		kind string
		px   int
	}{{"ic07", 128}, {"ic08", 256}, {"ic09", 512}} {
		p := encode(e.px)
		body.WriteString(e.kind)
		binary.Write(&body, binary.BigEndian, uint32(8+len(p)))
		body.Write(p)
	}
	var out bytes.Buffer
	out.WriteString("icns")
	binary.Write(&out, binary.BigEndian, uint32(8+body.Len()))
	out.Write(body.Bytes())
	return out.Bytes()
}

func main() {
	for name, data := range map[string][]byte{
		"roundhouse.png":  encode(256),
		"roundhouse.ico":  ico(16, 32, 48, 64, 128, 256),
		"roundhouse.icns": icns(),
	} {
		if err := os.WriteFile(name, data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

// ray is the distance from (x, y) to the part of the ray at angle i*45 degrees (screen coordinates) between
// the radii from and to, measured from the center of the icon.
func ray(x, y float64, i int, from, to float64) float64 {
	a := float64(i) * math.Pi / 4
	return segment(x, y, 128+from*math.Cos(a), 128+from*math.Sin(a), 128+to*math.Cos(a), 128+to*math.Sin(a))
}
