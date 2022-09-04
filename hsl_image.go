package image_utils

// This file contains an implementation of an HSL image, where each channel is
// 16-bit.

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Implements the color interface. Stores the H, S, and L components,
// respectively. *This will panic if the slice doesn't contain at least 3
// components.* Values after the first 3 are ignored. Each component is a
// fraction out of 0xffff.
type HSLColor []uint16

// Utility function to convert the 3 16-bit values to fractional components.
func (c HSLColor) HSLComponents() (float64, float64, float64) {
	h := float64(c[0]) / float64(0xffff)
	s := float64(c[1]) / float64(0xffff)
	l := float64(c[2]) / float64(0xffff)
	return h, s, l
}

func (c HSLColor) String() string {
	h, s, l := c.HSLComponents()
	return fmt.Sprintf("(%f, %f, %f)", h, s, l)
}

// Converts a given arbitrary RGB color to a single brightness value.
func convertToBrightness(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return float64(r+g+b) / (3.0 * 65535.0)
}

func clamp(v float64) float64 {
	if v <= 0.0 {
		return 0.0
	}
	if v >= 1.0 {
		return 1.0
	}
	return v
}

// Linearly maps a floating point value in [0, 1] to [0, 0xffff]. Clamps v to
// be in the range [0, 1].
func scaleTo16Bit(v float64) uint16 {
	return uint16(clamp(v) * float64(0xffff))
}

// Returns R, G, B, given a particular hue value.
func hueToRGB(h float64) (float64, float64, float64) {
	r := math.Abs((h*6.0)-3.0) - 1.0
	g := 2.0 - math.Abs((h*6.0)-2.0)
	b := 2.0 - math.Abs((h*6.0)-4.0)
	return clamp(r), clamp(g), clamp(b)
}

// I based this code off of the snippet here:
// https://gist.github.com/mathebox/e0805f72e7db3269ec22
func (c HSLColor) RGBA() (r, g, b, a uint32) {
	h, s, l := c.HSLComponents()
	r1, g1, b1 := hueToRGB(h)
	chroma := (1.0 - math.Abs(2.0*l-1)) * s
	r1 = (r1-0.5)*chroma + l
	g1 = (g1-0.5)*chroma + l
	b1 = (b1-0.5)*chroma + l
	r = uint32(scaleTo16Bit(r1))
	g = uint32(scaleTo16Bit(g1))
	b = uint32(scaleTo16Bit(b1))
	a = 0xffff
	return
}

// Implements the image interface. Internally uses HSL representation for each
// pixel.
type HSLImage struct {
	// We'll keep the HSL pixel data in a single slice to avoid any possible
	// padding if we use a slice of color structs instead. (This is why
	// HSLColor is a slice, rather than a struct.)
	Pixels []uint16
	W, H   int
}

func (h *HSLImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, h.W, h.H)
}

func (h *HSLImage) ColorModel() color.Model {
	return color.RGBA64Model
}

// Returns the HSLColor corresponding to the pixel at (x, y), or a separate,
// black, HSLColor if the coordinate is outside of the image boundaries.
func (h *HSLImage) HSLPixel(x, y int) HSLColor {
	if (x < 0) || (y < 0) || (x >= h.W) || (y >= h.H) {
		return HSLColor([]uint16{0, 0, 0})
	}
	i := 3 * (y*h.W + x)
	return HSLColor(h.Pixels[i : i+3])
}

func (h *HSLImage) At(x, y int) color.Color {
	return h.HSLPixel(x, y)
}

// Takes another image and sets a component of each of this image's pixels
// based on the brightness of each pixel in pic. The "componentOffset" must be
// 0 if setting hue, 1 if setting saturation, and 2 if setting luminosity.
func (h *HSLImage) SetComponent(pic image.Image, componentOffset int) error {
	if (componentOffset < 0) || (componentOffset > 2) {
		return fmt.Errorf("Invalid component offset: %d", componentOffset)
	}
	bounds := pic.Bounds().Canon()
	localX := 0
	localY := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		localX = 0
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			hslPixel := h.HSLPixel(localX, localY)
			// Convert the new component from the grayscale brightness of the
			// pixel in the source pic.
			newValue := scaleTo16Bit(convertToBrightness(pic.At(x, y)))
			hslPixel[componentOffset] = newValue
			localX++
		}
		localY++
	}
	return nil
}

// "Rotates" the hue value of each pixel in the image forward by the given
// amount.
func (h *HSLImage) AdjustHue(adjustment float64) {
	for y := 0; y < h.H; y++ {
		for x := 0; x < h.W; x++ {
			hslPixel := h.HSLPixel(x, y)
			// We'll just let this wrap around to take care of the rotation.
			hslPixel[0] += scaleTo16Bit(adjustment)
		}
	}
}

func NewHSLImage(w, h int) (*HSLImage, error) {
	if (w <= 0) || (h <= 0) {
		return nil, fmt.Errorf("Image bounds must be positive")
	}
	return &HSLImage{
		W:      w,
		H:      h,
		Pixels: make([]uint16, 3*w*h),
	}, nil
}

// TODO: Add a way to convert RGB to HSL color, and implement the Set(...)
// function for HSLImage.
