package image_utils

import (
	"fmt"
	"image"
	"image/color"
)

// Always will be a 1x1 pixel image. Can wrap an error message. Also satisfies
// the error interface.
type ErrorImage struct {
	Message error
}

func NewErrorImage(e error) *ErrorImage {
	return &ErrorImage{
		Message: e,
	}
}

func (pic *ErrorImage) Error() string {
	return pic.Message.Error()
}

func (pic *ErrorImage) Unwrap() error {
	return pic.Message
}

func (pic *ErrorImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, 1, 1)
}

func (pic *ErrorImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (pic *ErrorImage) At(x, y int) color.Color {
	return color.White
}

// Resizes an image by stretching or downsampling.
type ResizedImage struct {
	pic              image.Image
	w, h             int
	oldMinX, oldMinY int
	wRatio, hRatio   float64
}

// Returns a resized image, or an ErrorImage if the width or height are
// invalid.
func ResizeImage(in image.Image, w, h int) image.Image {
	if (w <= 0) || (h <= 0) {
		return NewErrorImage(fmt.Errorf("New image sizes must be positive"))
	}
	oldBounds := in.Bounds().Canon()
	wRatio := float64(oldBounds.Dx()) / float64(w)
	hRatio := float64(oldBounds.Dy()) / float64(h)
	return &ResizedImage{
		pic:     in,
		w:       w,
		h:       h,
		oldMinX: oldBounds.Min.X,
		oldMinY: oldBounds.Min.Y,
		wRatio:  wRatio,
		hRatio:  hRatio,
	}
}

func (r *ResizedImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, r.w, r.h)
}

func (r *ResizedImage) ColorModel() color.Model {
	return r.pic.ColorModel()
}

func (r *ResizedImage) At(x, y int) color.Color {
	return r.pic.At(int(r.wRatio*float64(x))+r.oldMinX,
		int(r.hRatio*float64(y))+r.oldMinY)
}

// Returns true if the two colors are exactly equal in the RGBA color space.
func ColorsEqual(a, b color.Color) bool {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return (r1 == r2) && (g1 == g2) && (b1 == b2) && (a1 == a2)
}

// Implements the color interface, but uses floating-point colors for easier
// multiplication. Does not include alpha for now.
type FloatColor struct {
	R float32
	G float32
	B float32
}

func (c FloatColor) Add(toAdd color.Color) FloatColor {
	converted := ConvertToFloatColor(toAdd)
	return FloatColor{
		R: c.R + converted.R,
		G: c.G + converted.G,
		B: c.B + converted.B,
	}
}

func (c FloatColor) Multiply(scale color.Color) FloatColor {
	converted := ConvertToFloatColor(scale)
	return FloatColor{
		R: c.R * converted.R,
		G: c.G * converted.G,
		B: c.B * converted.B,
	}
}

func (c FloatColor) Scale(scale float32) FloatColor {
	return FloatColor{
		R: c.R * scale,
		G: c.G * scale,
		B: c.B * scale,
	}
}

func (c FloatColor) RGBA() (r, g, b, a uint32) {
	var red, green, blue uint32
	if c.R >= 1.0 {
		red = 0xffff
	} else {
		red = uint32(c.R * float32(0xffff))
	}
	if c.G >= 1.0 {
		green = 0xffff
	} else {
		green = uint32(c.G * float32(0xffff))
	}
	if c.B >= 1.0 {
		blue = 0xffff
	} else {
		blue = uint32(c.B * float32(0xffff))
	}
	return red, green, blue, 0xffff
}

func (c FloatColor) String() string {
	return fmt.Sprintf("%04x%04x%04x", uint16(c.R*0xffff), uint16(c.G*0xffff),
		uint16(c.B*0xffff))
}

// Takes an arbitrary color and returns a FloatColor. Returns the original
// color if it's already a FloatColor, so be careful modifying what this
// returns.
func ConvertToFloatColor(c color.Color) FloatColor {
	tryResult, ok := c.(FloatColor)
	if ok {
		return tryResult
	}
	r, g, b, _ := c.RGBA()
	return FloatColor{
		R: float32(r) / 0xffff,
		G: float32(g) / 0xffff,
		B: float32(b) / 0xffff,
	}
}

// This implements the image.Image interface using FloatColor pixels.
type FloatColorImage struct {
	Pixels []FloatColor
	w, h   int
}

func (f *FloatColorImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, f.w, f.h)
}

func (f *FloatColorImage) ColorModel() color.Model {
	return color.ModelFunc(func(c color.Color) color.Color {
		return ConvertToFloatColor(c)
	})
}

func (f *FloatColorImage) At(x, y int) color.Color {
	if (x < 0) || (y < 0) || (x >= f.w) || (y >= f.h) {
		return color.Black
	}
	return f.Pixels[(y*f.w)+x]
}

// Adds a color to the given location in the FloatColorImage.
func (f *FloatColorImage) Add(x, y int, toAdd color.Color) {
	if (x < 0) || (y < 0) || (x >= f.w) || (y >= f.h) {
		return
	}
	pixel := f.Pixels[(y*f.w)+x]
	f.Pixels[(y*f.w)+x] = pixel.Add(toAdd)
}

// Creates a new blank FloatColorImage with the given dimensions.
func NewFloatColorImage(w, h int) (*FloatColorImage, error) {
	if (w <= 0) || (h <= 0) {
		return nil, fmt.Errorf("Image bounds must be positive")
	}
	return &FloatColorImage{
		w:      w,
		h:      h,
		Pixels: make([]FloatColor, w*h),
	}, nil
}

// Satisfies the Image interface, used to implement AddImageBorder.
type imageBorder struct {
	pic         image.Image
	picBounds   image.Rectangle
	borderWidth int
	fillColor   color.Color
}

func (b *imageBorder) ColorModel() color.Model {
	return b.pic.ColorModel()
}

func (b *imageBorder) Bounds() image.Rectangle {
	tmp := b.picBounds
	w := b.borderWidth * 2
	return image.Rect(0, 0, tmp.Dx()+w, tmp.Dy()+w)
}

func (b *imageBorder) At(x, y int) color.Color {
	tmp := b.picBounds
	if (x < b.borderWidth) || (y < b.borderWidth) {
		return b.fillColor
	}
	if (x >= tmp.Dx()+b.borderWidth) || (y >= tmp.Dy()+b.borderWidth) {
		return b.fillColor
	}
	return b.pic.At(x-b.borderWidth+tmp.Min.X, y-b.borderWidth+tmp.Min.Y)
}

// Returns a new image, consisting of the given image surrounded by a solid
// color border with the given color and width in pixels.
func AddImageBorder(pic image.Image, borderColor color.Color,
	width int) image.Image {
	return &imageBorder{
		pic:         pic,
		picBounds:   pic.Bounds().Canon(),
		borderWidth: width,
		fillColor:   borderColor,
	}
}
