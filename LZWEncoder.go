package gifencoder

/*
LZWEncoder.go

Authors
Kevin Weiner (original Java version - kweiner@fmsware.com)
Thibault Imbert (AS3 version - bytearray.org)
Johan Nordberg (JS version - code@johan-nordberg.com)

Acknowledgements
GIFCOMPR.C - GIF Image compression routines
Lempel-Ziv compression based on 'compress'. GIF modifications by
David Rowley (mgardi@watdcsu.waterloo.edu)
GIF Image compression - modified 'compress'
Based on: compress.c - File compression ala IEEE Computer, June 1984.
By Authors: Spencer W. Thomas (decvax!harpo!utah-cs!utah-gr!thomas)
Jim McKie (decvax!mcvax!jim)
Steve Davies (decvax!vax135!petsd!peora!srd)
Ken Turkowski (decvax!decwrl!turtlevax!ken)
James A. Woods (decvax!ihnp4!ames!jaw)
Joe Orost (decvax!vax135!petsd!joe)

(Go port 2024)
*/

const (
	EOF   = -1
	BITS  = 12
	HSIZE = 5003 // 80% occupancy
)

var masks = []int{
	0x0000, 0x0001, 0x0003, 0x0007, 0x000F, 0x001F,
	0x003F, 0x007F, 0x00FF, 0x01FF, 0x03FF, 0x07FF,
	0x0FFF, 0x1FFF, 0x3FFF, 0x7FFF, 0xFFFF,
}

// LZWEncoder encodes image data using LZW compression
type LZWEncoder struct {
	width        int
	height       int
	pixels       []byte
	initCodeSize int
	remaining    int
	curPixel     int
}

// NewLZWEncoder creates a new LZW encoder
func NewLZWEncoder(width, height int, pixels []byte, colorDepth int) *LZWEncoder {
	initCodeSize := colorDepth
	if initCodeSize < 2 {
		initCodeSize = 2
	}

	return &LZWEncoder{
		width:        width,
		height:       height,
		pixels:       pixels,
		initCodeSize: initCodeSize,
		remaining:    width * height,
		curPixel:     0,
	}
}

// Encode encodes and writes pixel data to the output stream
func (enc *LZWEncoder) Encode(out *ByteArray) {
	out.WriteByte(byte(enc.initCodeSize))  // write "initial code size" byte
	enc.remaining = enc.width * enc.height // reset navigation variables
	enc.curPixel = 0
	enc.compress(enc.initCodeSize+1, out) // compress and write the pixel data
	out.WriteByte(0)                      // write block terminator
}

// nextPixel returns the next pixel from the image
func (enc *LZWEncoder) nextPixel() int {
	if enc.remaining == 0 {
		return EOF
	}
	enc.remaining--
	pix := enc.pixels[enc.curPixel]
	enc.curPixel++
	return int(pix) & 0xff
}

// MAXCODE returns the maximum code value for n bits
func MAXCODE(nBits int) int {
	return (1 << nBits) - 1
}

// compress performs LZW compression
func (enc *LZWEncoder) compress(initBits int, out *ByteArray) {
	var (
		fcode    int
		c        int
		i        int
		ent      int
		disp     int
		hsizeReg int
		hshift   int
	)

	// 这些变量需要在闭包中共享和修改
	gInitBits := initBits
	clearFlg := false
	nBits := gInitBits
	maxcode := MAXCODE(nBits)

	clearCode := 1 << (initBits - 1)
	eofCode := clearCode + 1
	freeEnt := clearCode + 2

	aCount := 0
	curAccum := 0
	curBits := 0

	accum := make([]byte, 256)
	htab := make([]int, HSIZE)
	codetab := make([]int, HSIZE)

	// Flush the packet to disk, and reset the accumulator
	flushChar := func() {
		if aCount > 0 {
			out.WriteByte(byte(aCount))
			out.WriteBytes(accum[:aCount])
			aCount = 0
		}
	}

	// Add a character to the end of the current packet
	charOut := func(c byte) {
		accum[aCount] = c
		aCount++
		if aCount >= 254 {
			flushChar()
		}
	}

	// Clear out the hash table
	clHash := func(hsize int) {
		for i := 0; i < hsize; i++ {
			htab[i] = -1
		}
	}

	var output func(int)
	output = func(code int) {
		curAccum &= masks[curBits]

		if curBits > 0 {
			curAccum |= (code << curBits)
		} else {
			curAccum = code
		}

		curBits += nBits

		for curBits >= 8 {
			charOut(byte(curAccum & 0xff))
			curAccum >>= 8
			curBits -= 8
		}

		// If the next entry is going to be too big for the code size,
		// then increase it, if possible.
		if freeEnt > maxcode || clearFlg {
			if clearFlg {
				// 修复：先赋值再计算 maxcode
				nBits = gInitBits
				maxcode = MAXCODE(nBits)
				clearFlg = false
			} else {
				nBits++
				if nBits == BITS {
					maxcode = 1 << BITS
				} else {
					maxcode = MAXCODE(nBits)
				}
			}
		}

		if code == eofCode {
			// At EOF, write the rest of the buffer.
			for curBits > 0 {
				charOut(byte(curAccum & 0xff))
				curAccum >>= 8
				curBits -= 8
			}
			flushChar()
		}
	}

	// table clear for block compress
	clBlock := func() {
		clHash(HSIZE)
		freeEnt = clearCode + 2
		clearFlg = true
		output(clearCode)
	}

	// Set up the necessary values
	ent = enc.nextPixel()

	hshift = 0
	for fcode = HSIZE; fcode < 65536; fcode *= 2 {
		hshift++
	}
	hshift = 8 - hshift // set hash code range bound

	hsizeReg = HSIZE
	clHash(hsizeReg) // clear hash table

	output(clearCode)

outerLoop:
	for {
		c = enc.nextPixel()
		if c == EOF {
			break
		}

		fcode = (c << BITS) + ent
		i = (c << hshift) ^ ent // xor hashing

		if htab[i] == fcode {
			ent = codetab[i]
			continue
		} else if htab[i] >= 0 { // non-empty slot
			disp = hsizeReg - i // secondary hash (after G. Knott)
			if i == 0 {
				disp = 1
			}

			for {
				i -= disp
				if i < 0 {
					i += hsizeReg
				}

				if htab[i] == fcode {
					ent = codetab[i]
					continue outerLoop
				}

				if htab[i] < 0 {
					break
				}
			}
		}

		output(ent)
		ent = c

		if freeEnt < (1 << BITS) {
			codetab[i] = freeEnt // code -> hashtable
			freeEnt++
			htab[i] = fcode
		} else {
			clBlock()
		}
	}

	// Put out the final code.
	output(ent)
	output(eofCode)
}
