package gifencoder

import (
	"image"
	"image/color"
)

// GIFEncoder encodes images into GIF format
type GIFEncoder struct {
	// image size
	width  int
	height int

	// transparent color if given
	transparent *color.RGBA

	// transparent index in color table
	transIndex int

	// -1 = no repeat, 0 = forever. anything else is repeat count
	repeat int

	// frame delay (hundredths)
	delay int

	image         image.Image // current frame
	pixels        []byte      // RGB byte array from frame
	indexedPixels []byte      // converted frame indexed to palette
	colorDepth    int         // number of bit planes
	colorTab      []byte      // RGB palette
	neuQuant      *NeuQuant   // NeuQuant instance that was used to generate colorTab
	usedEntry     []bool      // active palette entries
	palSize       int         // color table size (bits-1)
	dispose       int         // disposal code (-1 = use default)
	firstFrame    bool
	sample        int          // default sample interval for quantizer
	ditherMethod  DitherMethod // dithering method
	serpentine    bool         // serpentine scanning for dithering
	globalPalette []byte

	out *ByteArray
}

// NewGIFEncoder creates a new GIF encoder
func NewGIFEncoder(width, height int) *GIFEncoder {
	return &GIFEncoder{
		width:        width,
		height:       height,
		repeat:       -1,
		delay:        0,
		dispose:      -1,
		firstFrame:   true,
		sample:       10,
		ditherMethod: DitherNone,
		serpentine:   false,
		palSize:      7,
		out:          NewByteArray(),
		usedEntry:    make([]bool, 256),
	}
}

// SetDelay sets the delay time between each frame, or changes it for subsequent frames
func (ge *GIFEncoder) SetDelay(milliseconds int) {
	ge.delay = milliseconds / 10
}

// SetFrameRate sets frame rate in frames per second
func (ge *GIFEncoder) SetFrameRate(fps int) {
	ge.delay = 100 / fps
}

// SetDispose sets the GIF frame disposal code
func (ge *GIFEncoder) SetDispose(disposalCode int) {
	if disposalCode >= 0 {
		ge.dispose = disposalCode
	}
}

// SetRepeat sets the number of times the set of GIF frames should be played
func (ge *GIFEncoder) SetRepeat(repeat int) {
	ge.repeat = repeat
}

// SetTransparent sets the transparent color
func (ge *GIFEncoder) SetTransparent(c *color.RGBA) {
	ge.transparent = c
}

// SetQuality sets quality of color quantization (1-30, lower is better)
func (ge *GIFEncoder) SetQuality(quality int) {
	if quality < 1 {
		quality = 1
	}
	ge.sample = quality
}

// SetDither sets dithering method. Available methods:
// - "none" or "" or false: no dithering
// - "FloydSteinberg" or true: Floyd-Steinberg dithering (recommended)
// - "FalseFloydSteinberg": False Floyd-Steinberg dithering
// - "Stucki": Stucki dithering
// - "Atkinson": Atkinson dithering
// Add "-serpentine" suffix to use serpentine scanning (e.g., "FloydSteinberg-serpentine")
func (ge *GIFEncoder) SetDither(method interface{}) {
	ge.serpentine = false

	switch v := method.(type) {
	case bool:
		if v {
			ge.ditherMethod = DitherFloydSteinberg
		} else {
			ge.ditherMethod = DitherNone
		}
	case string:
		// 检查是否有 serpentine 后缀
		if len(v) > 11 && v[len(v)-11:] == "-serpentine" {
			ge.serpentine = true
			v = v[:len(v)-11]
		}

		switch v {
		case "FloydSteinberg":
			ge.ditherMethod = DitherFloydSteinberg
		case "FalseFloydSteinberg":
			ge.ditherMethod = DitherFalseFloydSteinberg
		case "Stucki":
			ge.ditherMethod = DitherStucki
		case "Atkinson":
			ge.ditherMethod = DitherAtkinson
		case "none", "":
			ge.ditherMethod = DitherNone
		default:
			ge.ditherMethod = DitherNone
		}
	case DitherMethod:
		ge.ditherMethod = v
	default:
		ge.ditherMethod = DitherNone
	}
}

// SetGlobalPalette sets global palette for all frames
func (ge *GIFEncoder) SetGlobalPalette(palette []byte) {
	ge.globalPalette = palette
}

// GetGlobalPalette returns global palette used for all frames
func (ge *GIFEncoder) GetGlobalPalette() []byte {
	if ge.globalPalette != nil && len(ge.globalPalette) > 0 {
		result := make([]byte, len(ge.globalPalette))
		copy(result, ge.globalPalette)
		return result
	}
	return nil
}

// AddFrame adds next GIF frame
func (ge *GIFEncoder) AddFrame(img image.Image) error {
	ge.image = img

	if ge.globalPalette != nil && len(ge.globalPalette) > 0 {
		ge.colorTab = ge.globalPalette
	} else {
		ge.colorTab = nil
	}

	ge.getImagePixels() // convert to correct format if necessary
	ge.analyzePixels()  // build color table & map pixels

	if ge.firstFrame {
		ge.writeHeader()  // GIF header
		ge.writeLSD()     // logical screen descriptor
		ge.writePalette() // global color table
		if ge.repeat >= 0 {
			ge.writeNetscapeExt()
		}
	}

	ge.writeGraphicCtrlExt() // write graphic control extension
	ge.writeImageDesc()      // image descriptor

	if !ge.firstFrame && ge.globalPalette == nil {
		ge.writePalette() // local color table
	}

	ge.writePixels() // encode and write pixel data

	ge.firstFrame = false
	return nil
}

// Finish adds final trailer to the GIF stream
func (ge *GIFEncoder) Finish() {
	ge.out.WriteByte(0x3b) // gif trailer
}

// GetData retrieves the GIF stream as byte array
func (ge *GIFEncoder) GetData() []byte {
	return ge.out.GetData()
}

// Stream returns the output ByteArray
func (ge *GIFEncoder) Stream() *ByteArray {
	return ge.out
}

// writeHeader writes GIF file header
func (ge *GIFEncoder) writeHeader() {
	ge.out.WriteUTFBytes("GIF89a")
}

// analyzePixels analyzes current frame colors and creates color map
func (ge *GIFEncoder) analyzePixels() {
	if ge.colorTab == nil {
		ge.neuQuant = NewNeuQuant(ge.pixels, ge.sample)
		ge.neuQuant.BuildColormap() // create reduced palette
		ge.colorTab = ge.neuQuant.GetColormap()
	}

	// map image pixels to new palette
	if ge.ditherMethod != DitherNone {
		// 使用抖动
		ge.ditherPixels(ge.ditherMethod, ge.serpentine)
	} else {
		// 不使用抖动
		ge.indexPixels()
	}

	ge.pixels = nil
	ge.colorDepth = 8
	ge.palSize = 7

	// get closest match to transparent color if specified
	if ge.transparent != nil {
		ge.transIndex = ge.findClosest(*ge.transparent, true)
	}
}

// indexPixels indexes pixels without dithering
func (ge *GIFEncoder) indexPixels() {
	nPix := len(ge.pixels) / 3
	ge.indexedPixels = make([]byte, nPix)

	k := 0
	for j := 0; j < nPix; j++ {
		index := ge.findClosestRGB(
			ge.pixels[k]&0xff,
			ge.pixels[k+1]&0xff,
			ge.pixels[k+2]&0xff,
		)
		ge.usedEntry[index] = true
		ge.indexedPixels[j] = byte(index)
		k += 3
	}
}

// findClosest returns index of palette color closest to c
func (ge *GIFEncoder) findClosest(c color.RGBA, used bool) int {
	return ge.findClosestRGB(c.R, c.G, c.B)
}

// findClosestRGB finds closest RGB color in palette
func (ge *GIFEncoder) findClosestRGB(r, g, b byte) int {
	if ge.colorTab == nil {
		return -1
	}

	if ge.neuQuant != nil {
		return ge.neuQuant.LookupRGB(r, g, b)
	}

	minpos := 0
	dmin := 256 * 256 * 256
	length := len(ge.colorTab)

	for i, index := 0, 0; i < length; index++ {
		dr := int(r) - int(ge.colorTab[i])
		i++
		dg := int(g) - int(ge.colorTab[i])
		i++
		db := int(b) - int(ge.colorTab[i])
		i++

		d := dr*dr + dg*dg + db*db
		if d < dmin {
			dmin = d
			minpos = index
		}
	}

	return minpos
}

// getImagePixels extracts image pixels into byte array
func (ge *GIFEncoder) getImagePixels() {
	w := ge.width
	h := ge.height

	ge.pixels = make([]byte, w*h*3)

	bounds := ge.image.Bounds()

	minX := bounds.Min.X
	minY := bounds.Min.Y
	maxX := bounds.Max.X
	maxY := bounds.Max.Y

	availWidth := maxX - minX
	availHeight := maxY - minY

	if availWidth != w || availHeight != h {
		// 使用较小的尺寸避免越界
		if availWidth < w {
			w = availWidth
		}
		if availHeight < h {
			h = availHeight
		}
	}

	count := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := ge.image.At(minX+x, minY+y).RGBA()

			// RGBA() 返回 0-65535 的值，需要转换为 0-255
			ge.pixels[count] = byte(r >> 8)
			count++
			ge.pixels[count] = byte(g >> 8)
			count++
			ge.pixels[count] = byte(b >> 8)
			count++
		}
	}

	// 如果实际读取的像素少于预期，用黑色填充剩余部分
	expectedSize := ge.width * ge.height * 3
	for count < expectedSize {
		ge.pixels[count] = 255
		count++
	}
}

// writeGraphicCtrlExt writes Graphic Control Extension
func (ge *GIFEncoder) writeGraphicCtrlExt() {
	ge.out.WriteByte(0x21) // extension introducer
	ge.out.WriteByte(0xf9) // GCE label
	ge.out.WriteByte(4)    // data block size

	transp := 0
	disp := 0
	if ge.transparent == nil {
		transp = 0
		disp = 0 // dispose = no action
	} else {
		transp = 1
		disp = 2 // force clear if using transparent color
	}

	if ge.dispose >= 0 {
		disp = ge.dispose & 7 // user override
	}
	disp <<= 2

	// packed fields
	ge.out.WriteByte(byte(
		0 | // 1:3 reserved
			disp | // 4:6 disposal
			0 | // 7 user input - 0 = none
			transp, // 8 transparency flag
	))

	ge.writeShort(ge.delay)               // delay x 1/100 sec
	ge.out.WriteByte(byte(ge.transIndex)) // transparent color index
	ge.out.WriteByte(0)                   // block terminator
}

// writeImageDesc writes Image Descriptor
func (ge *GIFEncoder) writeImageDesc() {
	ge.out.WriteByte(0x2c) // image separator
	ge.writeShort(0)       // image position x,y = 0,0
	ge.writeShort(0)
	ge.writeShort(ge.width) // image size
	ge.writeShort(ge.height)

	// packed fields
	if ge.firstFrame || ge.globalPalette != nil {
		// no LCT - GCT is used for first (or only) frame
		ge.out.WriteByte(0)
	} else {
		// specify normal LCT
		ge.out.WriteByte(byte(
			0x80 | // 1 local color table 1=yes
				0 | // 2 interlace - 0=no
				0 | // 3 sorted - 0=no
				0 | // 4-5 reserved
				ge.palSize, // 6-8 size of color table
		))
	}
}

// writeLSD writes Logical Screen Descriptor
func (ge *GIFEncoder) writeLSD() {
	// logical screen size
	ge.writeShort(ge.width)
	ge.writeShort(ge.height)

	// packed fields
	ge.out.WriteByte(byte(
		0x80 | // 1 : global color table flag = 1 (gct used)
			0x70 | // 2-4 : color resolution = 7
			0x00 | // 5 : gct sort flag = 0
			ge.palSize, // 6-8 : gct size
	))

	ge.out.WriteByte(0) // background color index
	ge.out.WriteByte(0) // pixel aspect ratio - assume 1:1
}

// writeNetscapeExt writes Netscape application extension to define repeat count
func (ge *GIFEncoder) writeNetscapeExt() {
	ge.out.WriteByte(0x21)              // extension introducer
	ge.out.WriteByte(0xff)              // app extension label
	ge.out.WriteByte(11)                // block size
	ge.out.WriteUTFBytes("NETSCAPE2.0") // app id + auth code
	ge.out.WriteByte(3)                 // sub-block size
	ge.out.WriteByte(1)                 // loop sub-block id
	ge.writeShort(ge.repeat)            // loop count
	ge.out.WriteByte(0)                 // block terminator
}

// writePalette writes color table
func (ge *GIFEncoder) writePalette() {
	ge.out.WriteBytes(ge.colorTab)
	n := (3 * 256) - len(ge.colorTab)
	for i := 0; i < n; i++ {
		ge.out.WriteByte(0)
	}
}

// writeShort writes 16-bit value in little-endian order
func (ge *GIFEncoder) writeShort(value int) {
	ge.out.WriteByte(byte(value & 0xFF))
	ge.out.WriteByte(byte((value >> 8) & 0xFF))
}

// writePixels encodes and writes pixel data
func (ge *GIFEncoder) writePixels() {
	enc := NewLZWEncoder(ge.width, ge.height, ge.indexedPixels, ge.colorDepth)
	enc.Encode(ge.out)
}
