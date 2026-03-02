//go:build !linux
// +build !linux

package fb

import (
	"errors"
	"image"
	"image/color"
)

// Device is a non-functional framebuffer stub for non-Linux builds.
type Device struct {
	bounds image.Rectangle
}

func Open(_ string) (*Device, error) {
	return nil, errors.New("framebuffer is only supported on linux")
}

func (d *Device) Close() {}

func (d *Device) Bounds() image.Rectangle {
	if d == nil || d.bounds.Empty() {
		return image.Rect(0, 0, 0, 0)
	}
	return d.bounds
}

func (d *Device) ColorModel() color.Model {
	return color.RGBAModel
}

func (d *Device) At(_, _ int) color.Color {
	return color.RGBA{}
}

func (d *Device) Set(_, _ int, _ color.Color) {}
