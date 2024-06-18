package main

import (
	"github.com/d21d3q/framebuffer"
	"image"
	"os"
	// "image/color"
	"image/draw"
	"image/png"
	log "github.com/sirupsen/logrus"
)

func main() {
	fb, err := framebuffer.Open("/dev/fb0")
	if err != nil {
		panic(err)
	}
	defer fb.Close()

	infile, err := os.Open("/var/lib/gsplash/gsplash.png")
	if err != nil {
		log.WithError(err).Error("Failed to open image file")
		return
	}
	defer infile.Close()

	img, err := png.Decode(infile)
	if err != nil {
		log.WithError(err).Error("Failed to decode image")
		return
	}

	// magenta := image.NewUniform(color.RGBA{255, 0, 128, 255})
	draw.Draw(fb, fb.Bounds(), img, image.ZP, draw.Src)
}
