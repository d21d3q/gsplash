package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/d21d3q/gsplash/fb"
	log "github.com/sirupsen/logrus"
)

const (
	defaultFBDevice = "/dev/fb0"
	defaultImage    = "/usr/share/gsplash/gsplash.png"
)

func main() {
	if err := run(); err != nil {
		log.WithError(err).Error("gsplash failed")
		os.Exit(1)
	}
}

func run() error {
	var (
		fbDevice    = flag.String("device", defaultFBDevice, "framebuffer device")
		imagePath   = flag.String("image", defaultImage, "image path")
		background  = flag.String("background", "#000000", "background color in #RRGGBB or 0xRRGGBB format")
		waitSignal  = flag.Bool("wait", false, "wait for SIGINT/SIGTERM before exiting")
		clearOnly   = flag.Bool("clear", false, "clear the framebuffer with background color and exit")
		timeout     = flag.Duration("timeout", 20*time.Second, "how long to wait for framebuffer/image before failing")
		retryEvery  = flag.Duration("retry", 200*time.Millisecond, "retry interval while waiting for framebuffer/image")
		renderMode  = flag.String("mode", "contain", "render mode: contain|cover|center|stretch")
		clearOnExit = flag.Bool("clear-on-exit", false, "clear framebuffer on signal exit when --wait is enabled")
	)
	flag.Parse()

	bgColor, err := parseColor(*background)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	defer stop()

	fbDev, err := openFramebufferWithRetry(ctx, *fbDevice, *timeout, *retryEvery)
	if err != nil {
		return err
	}
	defer fbDev.Close()

	if *clearOnly {
		fillFramebuffer(fbDev, bgColor)
		return nil
	}

	img, err := decodeImageWithRetry(ctx, *imagePath, *timeout, *retryEvery)
	if err != nil {
		return err
	}

	renderToFramebuffer(fbDev, img, bgColor, *renderMode)

	if !*waitSignal {
		return nil
	}

	<-ctx.Done()
	if *clearOnExit {
		fillFramebuffer(fbDev, bgColor)
	}
	return nil
}

func decodeImageWithRetry(ctx context.Context, path string, timeout, interval time.Duration) (image.Image, error) {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	for {
		img, err := decodeImage(path)
		if err == nil {
			return img, nil
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out decoding image %q: %w", path, err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func decodeImage(path string) (image.Image, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	img, _, err := image.Decode(in)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func openFramebufferWithRetry(ctx context.Context, path string, timeout, interval time.Duration) (*fb.Device, error) {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	for {
		dev, err := fb.Open(path)
		if err == nil {
			return dev, nil
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out opening framebuffer %q: %w", path, err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func fillFramebuffer(dst draw.Image, c color.Color) {
	draw.Draw(dst, dst.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
}

func renderToFramebuffer(dst draw.Image, src image.Image, bg color.Color, mode string) {
	bounds := dst.Bounds()
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, image.NewUniform(bg), image.Point{}, draw.Src)

	target, ok := targetRect(bounds, src.Bounds(), mode)
	if ok {
		if target.Dx() == src.Bounds().Dx() && target.Dy() == src.Bounds().Dy() {
			draw.Draw(canvas, target, src, src.Bounds().Min, draw.Over)
		} else {
			scaleNearest(canvas, target, src)
		}
	}

	draw.Draw(dst, bounds, canvas, bounds.Min, draw.Src)
}

func targetRect(dst image.Rectangle, src image.Rectangle, mode string) (image.Rectangle, bool) {
	dw, dh := dst.Dx(), dst.Dy()
	sw, sh := src.Dx(), src.Dy()
	if dw <= 0 || dh <= 0 || sw <= 0 || sh <= 0 {
		return image.Rectangle{}, false
	}

	switch mode {
	case "stretch":
		return dst, true
	case "center":
		return centeredRect(dst, sw, sh), true
	case "cover":
		return fitRect(dst, sw, sh, true), true
	case "contain", "":
		return fitRect(dst, sw, sh, false), true
	default:
		return fitRect(dst, sw, sh, false), true
	}
}

func centeredRect(dst image.Rectangle, w, h int) image.Rectangle {
	x := dst.Min.X + (dst.Dx()-w)/2
	y := dst.Min.Y + (dst.Dy()-h)/2
	return image.Rect(x, y, x+w, y+h)
}

func fitRect(dst image.Rectangle, sw, sh int, cover bool) image.Rectangle {
	dw, dh := dst.Dx(), dst.Dy()

	w := dw
	h := (w * sh) / sw
	if (cover && h < dh) || (!cover && h > dh) {
		h = dh
		w = (h * sw) / sh
	}

	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	return centeredRect(dst, w, h)
}

func scaleNearest(dst *image.RGBA, dstRect image.Rectangle, src image.Image) {
	srcB := src.Bounds()
	srcW, srcH := srcB.Dx(), srcB.Dy()
	dstW, dstH := dstRect.Dx(), dstRect.Dy()
	if srcW <= 0 || srcH <= 0 || dstW <= 0 || dstH <= 0 {
		return
	}

	for y := 0; y < dstH; y++ {
		sy := srcB.Min.Y + (y*srcH)/dstH
		for x := 0; x < dstW; x++ {
			sx := srcB.Min.X + (x*srcW)/dstW
			dst.Set(dstRect.Min.X+x, dstRect.Min.Y+y, src.At(sx, sy))
		}
	}
}

func parseColor(value string) (color.RGBA, error) {
	s := strings.TrimSpace(strings.ToLower(value))
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(s, "0x")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid background color %q: expected 6 hex digits", value)
	}

	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, errors.New("invalid background color")
	}

	return color.RGBA{
		R: uint8((v >> 16) & 0xff),
		G: uint8((v >> 8) & 0xff),
		B: uint8(v & 0xff),
		A: 0xff,
	}, nil
}
