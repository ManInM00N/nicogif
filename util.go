package gifencoder

import (
	"errors"
	"image"
)

// EncodeGIF is a convenience function to quickly encode multiple images into a GIF
// images: slice of images to encode
// delays: slice of delays in milliseconds for each frame
func EncodeGIF(images []image.Image, delays []int) ([]byte, error) {
	if len(images) == 0 {
		return nil, errors.New("no images provided")
	}

	bounds := images[0].Bounds()
	encoder := NewGIFEncoder(bounds.Dx(), bounds.Dy())
	encoder.SetRepeat(0) // loop forever
	encoder.SetQuality(10)

	for i, img := range images {
		if i < len(delays) {
			encoder.SetDelay(delays[i])
		} else {
			encoder.SetDelay(100) // default 100ms
		}

		if err := encoder.AddFrame(img); err != nil {
			return nil, err
		}
	}

	encoder.Finish()
	return encoder.GetData(), nil
}

// EncodeGIFWithOptions provides more control over encoding options
type EncodeOptions struct {
	Width         int    // width of output GIF
	Height        int    // height of output GIF
	Repeat        int    // -1 = once, 0 = forever, >0 = count
	Quality       int    // 1-30, lower is better
	Dither        bool   // enable dithering
	GlobalPalette []byte // optional global palette
	Delays        []int  // delays in milliseconds
}

// EncodeGIFWithOptions encodes images with custom options
func EncodeGIFWithOptions(images []image.Image, opts EncodeOptions) ([]byte, error) {
	if len(images) == 0 {
		return nil, errors.New("no images provided")
	}

	width := opts.Width
	height := opts.Height
	if width == 0 || height == 0 {
		bounds := images[0].Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	encoder := NewGIFEncoder(width, height)

	// Set repeat
	if opts.Repeat != 0 {
		encoder.SetRepeat(opts.Repeat)
	} else {
		encoder.SetRepeat(0) // default to loop forever
	}

	// Set quality
	quality := opts.Quality
	if quality == 0 {
		quality = 10 // default
	}
	encoder.SetQuality(quality)

	// Set dither
	encoder.SetDither(opts.Dither)

	// Set global palette
	if opts.GlobalPalette != nil {
		encoder.SetGlobalPalette(opts.GlobalPalette)
	}

	// Add frames
	for i, img := range images {
		delay := 100 // default 100ms
		if i < len(opts.Delays) && opts.Delays[i] > 0 {
			delay = opts.Delays[i]
		}
		encoder.SetDelay(delay)

		if err := encoder.AddFrame(img); err != nil {
			return nil, err
		}
	}

	encoder.Finish()
	return encoder.GetData(), nil
}
