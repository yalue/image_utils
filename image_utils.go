// The image_utils package contains a variety of functions I've written across
// various projects using Go's image.Image interface.
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

func clamp32(v float32) float32 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func (c FloatColor) Brightness() float32 {
	return (clamp32(c.R) + clamp32(c.G) + clamp32(c.B)) / 3.0
}

func (c FloatColor) RGBA() (r, g, b, a uint32) {
	r = uint32(clamp32(c.R) * float32(0xffff))
	g = uint32(clamp32(c.G) * float32(0xffff))
	b = uint32(clamp32(c.B) * float32(0xffff))
	a = 0xffff
	return
}

func (c FloatColor) String() string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("%02x%02x%02x", r>>8, g>>8, b>>8)
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

func (f *FloatColorImage) Set(x, y int, c color.Color) {
	if (x < 0) || (y < 0) || (x >= f.w) || (y >= f.h) {
		return
	}
	f.Pixels[y*f.w+x] = ConvertToFloatColor(c)
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

// Implements the image.Image interface, wraps an underlying image but presents
// a version of it rotated to the right.
type rotatedRightImage struct {
	newBounds    image.Rectangle
	originalMaxY int
	pic          image.Image
}

func (r *rotatedRightImage) ColorModel() color.Model {
	return r.pic.ColorModel()
}

func (r *rotatedRightImage) Bounds() image.Rectangle {
	return r.newBounds
}

func (r *rotatedRightImage) At(x, y int) color.Color {
	return r.pic.At(y, r.originalMaxY-x)
}

// Takes an input image and returns a new image, consisting of the original
// rotated to the right by 90 degrees. May not work correctly if the original
// image's bounds don't start at (0, 0). Continues referring to the same
// original image.
func RotateRight(pic image.Image) image.Image {
	originalBounds := pic.Bounds().Canon()
	// NOTE: This only works if the original image starts at 0, 0.
	newBounds := image.Rect(0, 0, originalBounds.Dy(), originalBounds.Dx())
	return &rotatedRightImage{
		newBounds:    newBounds,
		originalMaxY: originalBounds.Max.Y - 1,
		pic:          pic,
	}
}

// Implements the image.Image interface, wraps an underlying image, but
// presents it vertically flipped.
type verticalFlippedImage struct {
	yOffset int
	pic     image.Image
}

func (v *verticalFlippedImage) ColorModel() color.Model {
	return v.pic.ColorModel()
}

func (v *verticalFlippedImage) Bounds() image.Rectangle {
	return v.pic.Bounds()
}

func (v *verticalFlippedImage) At(x, y int) color.Color {
	return v.pic.At(x, v.yOffset-y)
}

// Takes an image and returns a new image, consisting of the image flipped
// vertically. May not work correctly if the original image's bounds don't
// start at (0, 0). Continues referring to the same original image.
func VerticalFlip(pic image.Image) image.Image {
	return &verticalFlippedImage{
		yOffset: pic.Bounds().Canon().Max.Y - 1,
		pic:     pic,
	}
}

// Works the same as verticalFlippedImage
type horizontalFlippedImage struct {
	xOffset int
	pic     image.Image
}

func (h *horizontalFlippedImage) ColorModel() color.Model {
	return h.pic.ColorModel()
}

func (h *horizontalFlippedImage) Bounds() image.Rectangle {
	return h.pic.Bounds()
}

func (h *horizontalFlippedImage) At(x, y int) color.Color {
	return h.pic.At(h.xOffset-x, y)
}

func HorizontalFlip(pic image.Image) image.Image {
	return &horizontalFlippedImage{
		xOffset: pic.Bounds().Canon().Max.X - 1,
		pic:     pic,
	}
}

// Takes an arbitrary image and converts it to an RGBA image. Resets the top-
// left corner if the returned image to be at 0, 0.
func ToRGBA(pic image.Image) *image.RGBA {
	b := pic.Bounds().Canon()
	w := b.Dx()
	h := b.Dy()
	toReturn := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			toReturn.Set(x, y, pic.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return toReturn
}

// Satisfies the Color interface. Used for grayscale float images.
type FloatGrayscale float32

func (f FloatGrayscale) RGBA() (r, g, b, a uint32) {
	v := uint32(clamp32(float32(f)) * float32(0xffff))
	return v, v, v, 0xffff
}

func ConvertToFloatGrayscale(c color.Color) FloatGrayscale {
	fg, isFloatGrayscale := c.(FloatGrayscale)
	if isFloatGrayscale {
		return fg
	}
	r, g, b, _ := c.RGBA()
	v := float32(r+g+b) / float32(0xffff*3)
	return FloatGrayscale(clamp32(v))
}

type FloatGrayscaleImage struct {
	W, H   int
	Pixels []float32
}

func (f *FloatGrayscaleImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, f.W, f.H)
}

func (f *FloatGrayscaleImage) ColorModel() color.Model {
	return color.ModelFunc(func(c color.Color) color.Color {
		return ConvertToFloatGrayscale(c)
	})
}

func (f *FloatGrayscaleImage) At(x, y int) color.Color {
	if (x < 0) || (x >= f.W) || (y < 0) || (y >= f.H) {
		return FloatGrayscale(0)
	}
	return FloatGrayscale(f.Pixels[y*f.W+x])
}

func (f *FloatGrayscaleImage) Set(x, y int, c color.Color) {
	if (x < 0) || (x >= f.W) || (y < 0) || (y >= f.H) {
		return
	}
	f.Pixels[y*f.W+x] = float32(ConvertToFloatGrayscale(c))
}

// Creates and returns an empty FloatGrayscaleImage.
func NewFloatGrayscaleImage(w, h int) (*FloatGrayscaleImage, error) {
	if (w <= 0) || (h <= 0) || ((w * h) <= 0) {
		return nil, fmt.Errorf("Invalid image dimensions (%dx%d=%d pixels)",
			w, h, w*h)
	}
	return &FloatGrayscaleImage{
		W:      w,
		H:      h,
		Pixels: make([]float32, w*h),
	}, nil
}
