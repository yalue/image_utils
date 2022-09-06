package image_utils

// This file contains the definition for the CompositeImage type.

import (
	"image"
	"image/color"
)

// This satisfies the Image interface, but wraps a slice of images as if they
// are "layers."  Images on higher layers are combined with lower layers using
// alpha blending, unless some upper pixel is fully opaque. Not particularly
// efficient for a large number of images; rasterizing is recommended for such
// cases. Boundaries are automatically resized to fully contain the bounding
// rects of any image that's contained.
type CompositeImage struct {
	// The images, with layer 0 being the bottom.
	layerPics []image.Image
	// The top-left points of each image, with point 0 being the top-left of
	// the bottom image.
	topLeftPoints []image.Point
	// The bounding rectangles of each image, converted into the coordinates of
	// the composite image.
	compositeBounds []image.Rectangle
	// Automatically adjusted as more images are added.
	bounds image.Rectangle
}

// Returns a new CompositeImage, that is empty.
func NewCompositeImage() *CompositeImage {
	return &CompositeImage{
		layerPics:       make([]image.Image, 0, 8),
		topLeftPoints:   make([]image.Point, 0, 8),
		compositeBounds: make([]image.Rectangle, 0, 8),
		bounds:          image.Rect(0, 0, 1, 1),
	}
}

func (c *CompositeImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.bounds.Dx(), c.bounds.Dy())
}

func (c *CompositeImage) ColorModel() color.Model {
	return color.RGBAModel
}

func (c *CompositeImage) At(x, y int) color.Color {
	pt := image.Pt(c.bounds.Min.X+x, c.bounds.Min.Y+y)
	if !pt.In(c.bounds) {
		return color.Transparent
	}
	for i := len(c.layerPics) - 1; i >= 0; i-- {
		if !pt.In(c.compositeBounds[i]) {
			continue
		}
		offset := c.topLeftPoints[i]
		v := c.layerPics[i].At(pt.X-offset.X, pt.Y-offset.Y)
		_, _, _, a := v.RGBA()
		if a >= 0xff00 {
			// If this color is fully opaque, then we don't need to look any
			// farther.
			return v
		}
		// TODO: Alpha-blend composite images. For now, we'll just only treat
		// things as fully opaque or fully transparent.
		if a != 0 {
			// TEMPORARY: Anything not fully transparent is treated as opaque.
			return v
		}
		// At this point, the color is treated as fully transparent, so move on
		// to the next image that could contain the point.
	}
	// We didn't hit any images with this point.
	return color.Transparent
}

// Adds a new "layer" to the composite image, consisting of the entire provided
// image, with its top-left corner set to the given point.
func (c *CompositeImage) AddImage(pic image.Image, topLeft image.Point) error {
	if topLeft.X < c.bounds.Min.X {
		c.bounds.Min.X = topLeft.X
	}
	if topLeft.Y < c.bounds.Min.Y {
		c.bounds.Min.Y = topLeft.Y
	}
	bounds := pic.Bounds().Canon()
	// TODO: This won't be quite right for images that don't start at 0, 0.
	bottomRight := image.Pt(topLeft.X+bounds.Dx(), topLeft.Y+bounds.Dy())
	if bottomRight.X > c.bounds.Max.X {
		c.bounds.Max.X = bottomRight.X
	}
	if bottomRight.Y > c.bounds.Max.Y {
		c.bounds.Max.Y = bottomRight.Y
	}
	compositeBounds := image.Rectangle{
		Min: topLeft,
		Max: bottomRight,
	}
	c.layerPics = append(c.layerPics, pic)
	c.topLeftPoints = append(c.topLeftPoints, topLeft)
	c.compositeBounds = append(c.compositeBounds, compositeBounds)
	return nil
}
