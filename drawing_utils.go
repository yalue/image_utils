package image_utils

// This file contains functions used for drawing to any type of image that
// supports the Set(...) function.

import (
	"fmt"
	"image"
	"image/color"
	"math"
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

// Returns a ShapeWalker that produces a sequence of points from a to b.
func GetLineWalker(a, b image.Point) ShapeWalker {
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
	lineWalker := GetLineWalker(a, b)
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

// Implements the ShapeWalker interface for the outline of a rectangle.
type rectangleWalker struct {
	edges []ShapeWalker
	// An index into the "edges" list, or 4 if done.
	segment int
}

// Returns a ShapeWalker that traverses the outline of the given rectangle.
func GetRectangleWalker(r image.Rectangle) ShapeWalker {
	r = r.Canon()
	topLeft := r.Min
	topRight := image.Pt(r.Max.X, r.Min.Y)
	bottomRight := r.Max
	bottomLeft := image.Pt(r.Min.X, r.Max.Y)
	edges := make([]ShapeWalker, 4)
	edges[0] = GetLineWalker(topLeft, topRight)
	edges[1] = GetLineWalker(topRight, bottomRight)
	edges[2] = GetLineWalker(bottomRight, bottomLeft)
	edges[3] = GetLineWalker(bottomLeft, topLeft)
	return &rectangleWalker{
		edges:   edges,
		segment: 0,
	}
}

func (w *rectangleWalker) Reset() {
	for i := range w.edges {
		w.edges[i].Reset()
	}
	w.segment = 0
}

func (w *rectangleWalker) Done() bool {
	return w.segment >= len(w.edges)
}

func (w *rectangleWalker) Next() image.Point {
	if w.edges[w.segment].Done() {
		w.segment++
	}
	if w.segment >= len(w.edges) {
		// Arbitrarily continue along the final segment if the caller ignored
		// the fact that we're done.
		return w.edges[len(w.edges)-1].Next()
	}
	return w.edges[w.segment].Next()
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

// Draws the shape described by w using color c, returning a new image
// containing the entire shape, centered. Returns an error if the walker is
// invalid or describes an invalid shape.
func DrawWalker(w ShapeWalker, c color.Color) (*image.RGBA, error) {
	minX := math.MaxInt
	maxX := math.MinInt
	minY := math.MaxInt
	maxY := math.MinInt
	// First, we'll just find the bounds
	w.Reset()
	for !w.Done() {
		p := w.Next()
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if maxX < minX {
		return nil, fmt.Errorf("No points were in the shape")
	}
	toReturn := image.NewRGBA(image.Rect(0, 0, maxX-minX, maxY-minY))
	w.Reset()
	for !w.Done() {
		p := w.Next()
		toReturn.Set(p.X-minX, p.Y-minY, c)
	}
	return toReturn, nil
}

// Holds data associated with pixels used when running the Jump-Flooding
// algorithm for approximating a Voronoi fill. Implements the color interface
// so that VoronoiImages can use it as a Color.
type VoronoiPixel struct {
	// This must be true if this pixel is one of the original "seed" colors.
	IsSeed bool
	// This must be false if the pixel has yet to be set to a seed color.
	IsDefined bool
	// The pixel's current color.
	Color color.Color
	// If this pixel is defined but not a seed, this will contain the location
	// of its current seed pixel. If it is a seed, this contains its own
	// location.
	SeedLocation image.Point
}

// Returned by VoronoiImage.At(...) for all cases when the pixel is out of
// bounds. (This allows us to avoid re-allocating multiple OOB pixels, and
// re-use the same copy for all out-of-bounds calls to At().)
var undefinedPixel = VoronoiPixel{
	IsSeed:    false,
	IsDefined: false,
}

// Used by the Voronoi-fill algorithms as a distance to undefined pixels.
// Should always be larger than the square of the largest expected distance.
const arbitraryLargeFloat = 1.0e12

func (p *VoronoiPixel) RGBA() (r, g, b, a uint32) {
	if !p.IsDefined {
		return 0, 0, 0, 0xffff
	}
	return p.Color.RGBA()
}

// Holds intermediate data for running the Voronoi fill algorithm. Supports the
// image interface itself, too. May be used directly by initializing the struct
// and calling JumpFloodFill, or used implicitly by calling the VoronoiFill(..)
// function on an arbitrary drawable image.
type VoronoiImage struct {
	// The width and height of the image
	W, H int
	// Must contain WxH VoronoiPixels
	Pixels []VoronoiPixel
}

func (v *VoronoiImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, v.W, v.H)
}

func (v *VoronoiImage) ColorModel() color.Model {
	return color.RGBAModel
}

// Returns an undefined VoronoiPixel for any out-of-bounds pixel. (This
// simplifies the flood fill code.)
func (v *VoronoiImage) At(x, y int) *VoronoiPixel {
	w := v.W
	h := v.H
	if (x < 0) || (x >= w) || (y < 0) || (y >= h) {
		return &undefinedPixel
	}
	return &(v.Pixels[y*w+x])
}

// Called as part of the inner loop for the jump fill algorithm. Checks the
// neighbors of the pixel at x, y that are exactly the given stepSize away.
func (v *VoronoiImage) setSinglePixel(x, y, stepSize int) {
	p := &(v.Pixels[y*v.W+x])
	if p.IsSeed {
		return
	}
	var curSeedDistance float32
	if p.IsDefined {
		dx := float32(x - p.SeedLocation.X)
		dy := float32(y - p.SeedLocation.Y)
		curSeedDistance = (dx * dx) + (dy * dy)
	}
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			if (i == 0) && (j == 0) {
				continue
			}
			target := v.At(x+(i*stepSize), y+(j*stepSize))
			if !target.IsDefined {
				// The target pixel doesn't have a color yet.
				continue
			}
			dx := float32(x - target.SeedLocation.X)
			dy := float32(y - target.SeedLocation.Y)
			distToTargetSeed := (dx * dx) + (dy * dy)
			if !p.IsDefined {
				// This is the first time this pixel has "seen" a defined
				// pixel.
				p.IsDefined = true
				p.Color = target.Color
				p.SeedLocation = target.SeedLocation
				curSeedDistance = distToTargetSeed
				continue
			}
			// Both us and the target already have colors.
			if distToTargetSeed >= curSeedDistance {
				continue
			}
			// The target's seed is closer to our current location than our
			// current seed, so take its color and location.
			p.SeedLocation = target.SeedLocation
			p.Color = target.Color
			curSeedDistance = distToTargetSeed
		}
	}
}

// Runs the Jump-Flod fill algorithm, returning an error if one occurs.
func (v *VoronoiImage) JumpFloodFill() error {
	w := v.W
	h := v.H
	stepSize := w
	if h > stepSize {
		stepSize = h
	}
	stepSize = stepSize >> 1
	for stepSize > 0 {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v.setSinglePixel(x, y, stepSize)
			}
		}
		stepSize = stepSize >> 1
	}
	// TODO: Try the additional 1-wide step before or after this loop, as
	// mentioned in the wikipedia article.
	return nil
}

// Fills any zero-valued pixel in pic using the values of the nonzero pixels
// using the Jump-Flooding Algorithm. See:
// https://en.wikipedia.org/wiki/Jump_flooding_algorithm
//
// May return an error, including if the input pic isn't a DrawableImage.
// Modifies the input image. Requires an isSeed function that returns true if
// called on a pixel coordinate that's a seed for the resulting image.
func VoronoiFill(m image.Image, isSeed func(x, y int) bool) error {
	pic, ok := m.(DrawableImage)
	if !ok {
		return fmt.Errorf("The given image isn't drawable")
	}
	bounds := pic.Bounds()
	if !((bounds.Min.X == 0) && (bounds.Min.Y == 0) && (bounds.Max.X > 0) &&
		(bounds.Max.Y > 0)) {
		return fmt.Errorf("Only images with bounds from 0,0 to positive X,Y " +
			"coordinates are supported")
	}

	// Initialize a VoronoiImage instance using the colors and seeds from the
	// given image.
	w := bounds.Dx()
	h := bounds.Dy()
	pixels := make([]VoronoiPixel, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !isSeed(x, y) {
				continue
			}
			i := y*w + x
			pixels[i].IsSeed = true
			pixels[i].IsDefined = true
			pixels[i].Color = m.At(x, y)
			pixels[i].SeedLocation = image.Pt(x, y)
		}
	}
	voronoiImage := &VoronoiImage{
		Pixels: pixels,
		W:      w,
		H:      h,
	}

	// Actually run the fill.
	e := voronoiImage.JumpFloodFill()
	if e != nil {
		return fmt.Errorf("Error running Voronoi fill: %w", e)
	}

	// Copy the results back into the original image
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pic.Set(x, y, voronoiImage.At(x, y))
		}
	}
	return nil
}

// Applies a blur to m with the given radius. Returns an error if the image
// isn't a DrawableImage.
func Blur(m image.Image, radius int) error {
	pic, ok := m.(DrawableImage)
	if !ok {
		return fmt.Errorf("The given image isn't drawable")
	}
	// TODO: Implement Blur
	return fmt.Errorf("Not yet implemented (blur on %v)", pic)
}
