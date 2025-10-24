package gifencoder

import (
	"errors"
	"image"
	"math"
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
	Width           int         // width of output GIF
	Height          int         // height of output GIF
	Repeat          int         // -1 = once, 0 = forever, >0 = count
	Quality         int         // 1-30, lower is better
	Dither          interface{} // dithering method: bool, string, or DitherMethod
	GlobalPalette   []byte      // optional global palette
	Delays          []int       // delays in milliseconds
	SaturationBoost float64     // 饱和度增强, [0.0,2.0], 1.0为原始
	ContrastBoost   float64     // 对比度增强, [0.0,2.0], 1.0为原始
}

func NewGIFEncoderWithOptions(width, height int, opts EncodeOptions) *GIFEncoder {
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
	if opts.Dither != nil {
		encoder.SetDither(opts.Dither)
	}

	// Set color enhancement
	opts.ContrastBoost = minFloat(2.0, maxFloat(1.0, opts.ContrastBoost))
	opts.SaturationBoost = minFloat(2.0, maxFloat(1.0, opts.SaturationBoost))
	encoder.SetColorEnhancement(opts.SaturationBoost, opts.ContrastBoost)

	// Set global palette
	if opts.GlobalPalette != nil {
		encoder.SetGlobalPalette(opts.GlobalPalette)
	}
	return encoder
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
	if opts.Dither != nil {
		encoder.SetDither(opts.Dither)
	}

	// Set color enhancement
	opts.ContrastBoost = minFloat(2.0, maxFloat(1.0, opts.ContrastBoost))
	opts.SaturationBoost = minFloat(2.0, maxFloat(1.0, opts.SaturationBoost))
	encoder.SetColorEnhancement(opts.SaturationBoost, opts.ContrastBoost)

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

// 辅助函数
func maxFloat(a ...float64) float64 {
	if len(a) == 0 {
		return math.MaxFloat64
	}
	max := a[0]
	for _, b := range a[1:] {
		if b > max {
			max = b
		}
	}
	return max
}

func minFloat(a ...float64) float64 {
	if len(a) == 0 {
		return -math.MaxFloat64
	}
	min := a[0]
	for _, b := range a[1:] {
		if b < min {
			min = b
		}
	}
	return min
}
func hslToRGB(h, s, l float64) (float64, float64, float64) {
	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		hue2rgb := func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6.0*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6.0
			}
			return p
		}

		var q float64
		if l < 0.5 {
			q = l * (1.0 + s)
		} else {
			q = l + s - l*s
		}
		p := 2.0*l - q

		r = hue2rgb(p, q, h+1.0/3.0)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3.0)
	}

	return r, g, b
}

func clampFloat(val float64) byte {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return byte(val)
}
