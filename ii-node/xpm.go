package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

func ParseXpm(lines []string) (*image.RGBA, int) {
	nr := 0
	var img *image.RGBA
	start := false
	cformat := false
	last := false
	pal := make(map[string]color.RGBA)
	var w, h, cols, cpp, y int
	for _, l := range lines {
		nr++
		if !start && !strings.HasPrefix(l, "/* XPM */") && !strings.HasPrefix(l, "! XPM2") {
			continue
		}
		if !start {
			start = true
			continue
		}
		if strings.Contains(l, "static char ") {
			cformat = true
			continue
		}
		if cformat {
			if strings.HasPrefix(l, "/*") || strings.HasPrefix(l, "//") {
				continue
			}
			if strings.HasSuffix(l, "};") {
				last = true
				l = strings.TrimRight(l, "};")
			}
			l = strings.TrimRight(l, ",")
			l = strings.Trim(l, "\"")
		}
		l = strings.Replace(l, "\t", " ", -1)
		if cols == 0 { /* desc line */
			dsc := strings.Split(l, " ")
			if len(dsc) != 4 {
				return nil, 0
			}
			if len, _ := fmt.Sscanf(dsc[0], "%d", &w); len != 1 {
				return nil, 0
			}
			if len, _ := fmt.Sscanf(dsc[1], "%d", &h); len != 1 {
				return nil, 0
			}
			if len, _ := fmt.Sscanf(dsc[2], "%d", &cols); len != 1 {
				return nil, 0
			}
			if len, _ := fmt.Sscanf(dsc[3], "%d", &cpp); len != 1 {
				return nil, 0
			}
			upLeft := image.Point{0, 0}
			lowRight := image.Point{w, h}
			img = image.NewRGBA(image.Rectangle{upLeft, lowRight})
			continue
		}
		if cols > 0 {
			dsc := strings.Split(l, " c ")
			if len(dsc) != 2 {
				return nil, 0
			}
			rgb := color.RGBA{A: 255}
			len, _ := fmt.Sscanf(dsc[1], "#%02x%02x%02x", &rgb.R, &rgb.G, &rgb.B)
			if len != 3 {
				if dsc[1] != "None" {
					return nil, 0
				}
				rgb.R, rgb.G, rgb.B, rgb.A = 255, 255, 255, 0
			}
			pal[dsc[0]] = rgb
			cols--
			if cols == 0 {
				cols = -1 // done
			}
			continue
		}
		if y >= h && strings.HasPrefix(l, "}") {
			break
		}
		// image data
		for i := 0; i < len(l); i += 1 {
			e := i + cpp
			if e > len(l) {
				return nil, 0
			}
			sym := l[i:e]
			rgba, ok := pal[sym]
			if !ok {
				return nil, 0
			}
			img.Set(i/cpp, y, color.RGBA(rgba))
		}
		y++
		if y >= h && (!cformat || last) {
			break
		}
	}
	if y < h {
		return nil, 0
	}
	return img, nr
}
