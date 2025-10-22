package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"gifencoder"
)

func main() {
	fmt.Println("GIF Encoder Examples")
	fmt.Println("====================")

	// Example 1: Simple animation
	fmt.Println("\n1. Creating simple animation...")
	if err := simpleAnimation(); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("✅ Created animation.gif")
	}

	// Example 2: Gradient animation
	fmt.Println("\n2. Creating gradient animation...")
	if err := gradientAnimation(); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("✅ Created gradient.gif")
	}

	// Example 3: Custom options
	fmt.Println("\n3. Creating with custom options...")
	if err := customOptions(); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("✅ Created custom.gif")
	}

	fmt.Println("\nAll done!")
}

// simpleAnimation creates a simple moving circle animation
func simpleAnimation() error {
	width, height := 200, 200
	frames := make([]image.Image, 0)

	// Create 10 frames with a moving red circle
	for i := 0; i < 10; i++ {
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		// White background
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.White)
			}
		}

		// Draw moving red circle
		centerX := 50 + i*15
		centerY := 100
		radius := 30

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dx := x - centerX
				dy := y - centerY
				if dx*dx+dy*dy <= radius*radius {
					img.Set(x, y, color.RGBA{255, 0, 0, 255})
				}
			}
		}

		frames = append(frames, img)
	}

	// Encode to GIF
	delays := make([]int, len(frames))
	for i := range delays {
		delays[i] = 100 // 100ms per frame
	}

	gifData, err := gifencoder.EncodeGIF(frames, delays)
	if err != nil {
		return err
	}

	return os.WriteFile("animation.gif", gifData, 0644)
}

// gradientAnimation creates a color gradient animation
func gradientAnimation() error {
	width, height := 200, 200
	frames := make([]image.Image, 20)

	for f := 0; f < 20; f++ {
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				r := uint8((x + f*10) % 256)
				g := uint8((y + f*10) % 256)
				b := uint8(200)
				img.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}

		frames[f] = img
	}

	// Use simple encoding
	delays := make([]int, len(frames))
	for i := range delays {
		delays[i] = 50 // 50ms = 20 FPS
	}

	gifData, err := gifencoder.EncodeGIF(frames, delays)
	if err != nil {
		return err
	}

	return os.WriteFile("gradient.gif", gifData, 0644)
}

// customOptions demonstrates advanced usage with custom options
func customOptions() error {
	width, height := 150, 150
	frames := make([]image.Image, 15)

	// Create a spinning square animation
	for f := 0; f < 15; f++ {
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		// Dark background
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.RGBA{20, 20, 40, 255})
			}
		}

		// Draw colored square
		size := 50
		offsetX := 50
		offsetY := 50

		hue := float64(f) / 15.0
		r, g, b := hsvToRGB(hue, 1.0, 1.0)

		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				img.Set(offsetX+x, offsetY+y, color.RGBA{r, g, b, 255})
			}
		}

		frames[f] = img
	}

	// Use custom options
	opts := gifencoder.EncodeOptions{
		Width:   width,
		Height:  height,
		Repeat:  0, // loop forever
		Quality: 5, // high quality
		Dither:  false,
		Delays:  []int{80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80, 80},
	}

	gifData, err := gifencoder.EncodeGIFWithOptions(frames, opts)
	if err != nil {
		return err
	}

	return os.WriteFile("custom.gif", gifData, 0644)
}

// hsvToRGB converts HSV color to RGB (h: 0-1, s: 0-1, v: 0-1)
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	if s == 0 {
		val := uint8(v * 255)
		return val, val, val
	}

	h = h * 6
	i := int(h)
	f := h - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))

	var r, g, b float64
	switch i {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}

	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}
