package image_utils

// This file contains functions used for drawing to any type of image that
// supports the Set(...) function.

import (
	"fmt"
	"image"
	"image/color"
)

// This supersedes the image interface, as it also includes the function for
// drawing at a given pixel. Supported by most image types provided by Go's
// standard library.
type DrawableImage interface {
	image.Image
	Set(x, y int, c color.Color)
}

// Used when drawing to an image. Start by calling Reset(), and then call
// Next() to obtain subsequent points to shade, stopping when Done() returns
// true.
type ShapeWalker interface {
	Next() image.Point
	Done() bool
	Reset()
}

// Satisfies the ShapeWalker interface. Used when drawing lines that are larger
// in the horizontal axis.
type widerLine struct {
	origX int
	x     int
	y     float64
	maxX  int
	slope float64
}

func (l *widerLine) Next() image.Point {
	toReturn := image.Point{
		X: l.x,
		Y: int(l.y + (float64(l.x) * l.slope)),
	}
	l.x++
	return toReturn
}

func (l *widerLine) Done() bool {
	return l.x >= l.maxX
}

func (l *widerLine) Reset() {
	l.x = l.origX
}

// Satisfies the ShapeWalker interface. Used when drawing lines that are wider
// in the vertical axis.
type tallerLine struct {
	x     float64
	origY int
	y     int
	maxY  int
	slope float64
}

func (l *tallerLine) Next() image.Point {
	toReturn := image.Point{
		X: int(l.x + (float64(l.y) * l.slope)),
		Y: l.y,
	}
	l.y++
	return toReturn
}

func (l *tallerLine) Reset() {
	l.y = l.origY
}

func (l *tallerLine) Done() bool {
	return l.y >= l.maxY
}

func getLineWalker(a, b image.Point) ShapeWalker {
	dx := b.X - a.X
	dy := b.Y - a.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	// Always increment along the longer component of the line otherwise pixels
	// can be missed.
	if dx > dy {
		// Lines are always drawn from lower to higher pixel indices.
		if a.X > b.X {
			a, b = b, a
			// Recalculate the slope if we swapped the points around.
			dx = b.X - a.X
			dy = b.Y - a.Y
		}
		return &widerLine{
			x:     a.X,
			origX: a.X,
			y:     float64(a.Y),
			slope: float64(dy) / float64(dx),
			maxX:  b.X,
		}
	}
	if a.Y > b.Y {
		a, b = b, a
		dx = b.X - a.X
		dy = b.Y - a.Y
	}
	return &tallerLine{
		x:     float64(a.X),
		origY: a.Y,
		y:     a.Y,
		slope: float64(dx) / float64(dy),
		maxY:  b.Y,
	}
}

// Draws a line with the given color to the dst image. Returns an error if one
// occurs.
func DrawLine(a, b image.Point, c color.Color, dst DrawableImage) error {
	// Arbitrarily limit lines to 200 million pixels as a sanity check.
	maxSteps := 200000000
	lineWalker := getLineWalker(a, b)
	lineWalker.Reset()
	step := 0
	for !lineWalker.Done() {
		step++
		if step >= maxSteps {
			return fmt.Errorf("Tried drawing a line that was too long")
		}
		loc := lineWalker.Next()
		dst.Set(loc.X, loc.Y, c)
	}
	return nil
}

// Returns an image containing a picture of an upward-pointing arrow with the
// given color. The image may be an arbitrary, small, resolution, so use
// the ResizeImage function if you want it of a specific resolution. The
// image's background will be transparent.
func UpArrow(c color.Color) image.Image {
	t := color.RGBA{0, 0, 0, 0}
	cols := 9
	rows := 10
	toReturn := image.NewRGBA(image.Rect(0, 0, cols, rows))
	colors := []color.Color{
		t, t, t, t, t, t, t, t, t,
		t, t, t, t, c, t, t, t, t,
		t, t, t, c, c, c, t, t, t,
		t, t, c, c, c, c, c, t, t,
		t, c, c, c, c, c, c, c, t,
		t, t, t, c, c, c, t, t, t,
		t, t, t, c, c, c, t, t, t,
		t, t, t, c, c, c, t, t, t,
		t, t, t, c, c, c, t, t, t,
		t, t, t, t, t, t, t, t, t,
	}
	i := 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			toReturn.Set(x, y, colors[i])
			i++
		}
	}
	return toReturn
}

// Returns an image containing a picture of a right-pointing arrow with the
// given color. Works similarly to the UpArrow function.
func RightArrow(c color.Color) image.Image {
	return RotateRight(UpArrow(c))
}

// Returns an image containing a picture of a downward-pointing arrow with the
// given color. Works similarly to the UpArrow function.
func DownArrow(c color.Color) image.Image {
	return VerticalFlip(UpArrow(c))
}

// Returns an image containing a picture of a left-pointing arrow with the
// given color. Works similarly to the UpArrow function.
func LeftArrow(c color.Color) image.Image {
	return HorizontalFlip(RotateRight(UpArrow(c)))
}
