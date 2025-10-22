package gifencoder

import (
	"image"
	"image/color"
	_ "image/jpeg" // 注册 JPEG 解码器
	_ "image/png"  // 注册 PNG 解码器
	"os"
	"testing"
)

func TestNewGIFEncoder(t *testing.T) {
	encoder := NewGIFEncoder(100, 100)
	if encoder == nil {
		t.Fatal("NewGIFEncoder returned nil")
	}
	if encoder.width != 100 || encoder.height != 100 {
		t.Errorf("Expected dimensions 100x100, got %dx%d", encoder.width, encoder.height)
	}
}

func TestSetDelay(t *testing.T) {
	encoder := NewGIFEncoder(100, 100)
	encoder.SetDelay(500)
	if encoder.delay != 50 { // 500ms / 10 = 50
		t.Errorf("Expected delay 50, got %d", encoder.delay)
	}
}

func TestSetFrameRate(t *testing.T) {
	encoder := NewGIFEncoder(100, 100)
	encoder.SetFrameRate(10)
	if encoder.delay != 10 { // 100 / 10 = 10
		t.Errorf("Expected delay 10, got %d", encoder.delay)
	}
}

func TestByteArray(t *testing.T) {
	ba := NewByteArray()

	// Test writing single bytes
	for i := 0; i < 10; i++ {
		ba.WriteByte(byte(i))
	}

	data := ba.GetData()
	if len(data) != 10 {
		t.Errorf("Expected length 10, got %d", len(data))
	}

	for i := 0; i < 10; i++ {
		if data[i] != byte(i) {
			t.Errorf("Expected byte %d at index %d, got %d", i, i, data[i])
		}
	}
}

func TestByteArrayMultiplePages(t *testing.T) {
	ba := NewByteArray()

	// Write more than one page
	numBytes := ba.pageSize*2 + 100
	for i := 0; i < numBytes; i++ {
		ba.WriteByte(byte(i % 256))
	}

	data := ba.GetData()
	if len(data) != numBytes {
		t.Errorf("Expected length %d, got %d", numBytes, len(data))
	}
}

func TestNeuQuant(t *testing.T) {
	// Create a simple RGB pixel array
	pixels := make([]byte, 300) // 100 pixels * 3 channels
	for i := 0; i < len(pixels); i += 3 {
		pixels[i] = byte(i % 256)         // R
		pixels[i+1] = byte((i + 1) % 256) // G
		pixels[i+2] = byte((i + 2) % 256) // B
	}

	nq := NewNeuQuant(pixels, 10)
	nq.BuildColormap()

	colormap := nq.GetColormap()
	if len(colormap) != 256*3 {
		t.Errorf("Expected colormap length 768, got %d", len(colormap))
	}

	// Test lookup
	index := nq.LookupRGB(255, 0, 0)
	if index < 0 || index >= 256 {
		t.Errorf("Invalid color index: %d", index)
	}
}

func TestEncodeSimpleGIF(t *testing.T) {
	// Create a simple 10x10 red image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	encoder := NewGIFEncoder(10, 10)
	err := encoder.AddFrame(img)
	if err != nil {
		t.Fatalf("AddFrame failed: %v", err)
	}

	encoder.Finish()
	data := encoder.GetData()

	// Check GIF header
	if len(data) < 6 {
		t.Fatal("GIF data too short")
	}
	if string(data[0:6]) != "GIF89a" {
		t.Errorf("Invalid GIF header: %s", string(data[0:6]))
	}

	// Check trailer
	if data[len(data)-1] != 0x3b {
		t.Error("Missing GIF trailer")
	}
}

func TestEncodeMultiFrameGIF(t *testing.T) {
	frames := make([]image.Image, 3)
	colors := []color.RGBA{
		{255, 0, 0, 255}, // Red
		{0, 255, 0, 255}, // Green
		{0, 0, 255, 255}, // Blue
	}

	for i := 0; i < 3; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 20, 20))
		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				img.Set(x, y, colors[i])
			}
		}
		frames[i] = img
	}

	delays := []int{100, 100, 100}
	gifData, err := EncodeGIF(frames, delays)
	if err != nil {
		t.Fatalf("EncodeGIF failed: %v", err)
	}

	// Verify GIF structure
	if len(gifData) < 100 {
		t.Fatal("GIF data too short")
	}
	if string(gifData[0:6]) != "GIF89a" {
		t.Error("Invalid GIF header")
	}
}

func TestLZWEncoder(t *testing.T) {
	// Create simple test data
	pixels := make([]byte, 100)
	for i := range pixels {
		pixels[i] = byte(i % 10)
	}

	encoder := NewLZWEncoder(10, 10, pixels, 8)
	out := NewByteArray()
	encoder.Encode(out)

	data := out.GetData()
	if len(data) == 0 {
		t.Error("LZW encoder produced no output")
	}

	// Check initial code size
	if data[0] != 8 {
		t.Errorf("Expected initial code size 8, got %d", data[0])
	}
}

// Benchmark tests
func BenchmarkNeuQuant(b *testing.B) {
	pixels := make([]byte, 100*100*3)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nq := NewNeuQuant(pixels, 10)
		nq.BuildColormap()
	}
}

func BenchmarkEncodeFrame(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{
				uint8(x * 255 / 100),
				uint8(y * 255 / 100),
				128,
				255,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder := NewGIFEncoder(100, 100)
		encoder.AddFrame(img)
		encoder.Finish()
		_ = encoder.GetData()
	}
}

// Integration test - creates actual GIF file
func TestCreateActualGIF(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create animated GIF with gradient
	frames := make([]image.Image, 10)
	for f := 0; f < 10; f++ {
		img := image.NewRGBA(image.Rect(0, 0, 50, 50))
		for y := 0; y < 50; y++ {
			for x := 0; x < 50; x++ {
				r := uint8((x + f*5) % 256)
				g := uint8((y + f*5) % 256)
				b := uint8(200)
				img.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}
		frames[f] = img
	}

	delays := make([]int, 10)
	for i := range delays {
		delays[i] = 100
	}

	gifData, err := EncodeGIF(frames, delays)
	if err != nil {
		t.Fatalf("EncodeGIF failed: %v", err)
	}

	// Save to file
	filename := "test_output.gif"
	err = os.WriteFile(filename, gifData, 0644)
	if err != nil {
		t.Fatalf("Failed to write GIF file: %v", err)
	}
	defer os.Remove(filename)

	// Verify file exists and has reasonable size
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat GIF file: %v", err)
	}

	if info.Size() < 100 {
		t.Errorf("GIF file too small: %d bytes", info.Size())
	}

	t.Logf("Created test GIF: %s (%d bytes)", filename, info.Size())
}

func TestTransparentColor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	encoder := NewGIFEncoder(10, 10)
	encoder.SetTransparent(&color.RGBA{255, 255, 255, 255})

	err := encoder.AddFrame(img)
	if err != nil {
		t.Fatalf("AddFrame with transparent color failed: %v", err)
	}

	encoder.Finish()
	data := encoder.GetData()
	if len(data) == 0 {
		t.Error("No data generated with transparent color")
	}
}

func TestEncodeWithOptions(t *testing.T) {
	// Create test frames
	frames := make([]image.Image, 3)
	for i := 0; i < 3; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 20, 20))
		for y := 0; y < 20; y++ {
			for x := 0; x < 20; x++ {
				img.Set(x, y, color.RGBA{uint8(i * 85), 0, 0, 255})
			}
		}
		frames[i] = img
	}

	opts := EncodeOptions{
		Width:   20,
		Height:  20,
		Repeat:  0,
		Quality: 1,
		Delays:  []int{100, 100, 100},
	}

	gifData, err := EncodeGIFWithOptions(frames, opts)
	if err != nil {
		t.Fatalf("EncodeGIFWithOptions failed: %v", err)
	}

	if len(gifData) < 100 {
		t.Error("Generated GIF data too small")
	}
}
