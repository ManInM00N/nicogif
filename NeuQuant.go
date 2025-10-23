package gifencoder

/*
NeuQuant Neural-Net Quantization Algorithm
------------------------------------------

Copyright (c) 1994 Anthony Dekker

NEUQUANT Neural-Net quantization algorithm by Anthony Dekker, 1994.
See "Kohonen neural networks for optimal colour quantization"
in "Network: Computation in Neural Systems" Vol. 5 (1994) pp 351-367.
for a discussion of the algorithm.
See also http://members.ozemail.com.au/~dekker/NEUQUANT.HTML

Any party obtaining a copy of these files from the author, directly or
indirectly, is granted, free of charge, a full and unrestricted irrevocable,
world-wide, paid up, royalty-free, nonexclusive right and license to deal
in this software and documentation files (the "Software"), including without
limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
and/or sell copies of the Software, and to permit persons who receive
copies from any such party to do so, with the only requirement being
that this copyright notice remain intact.

(Go port 2024)
*/

const (
	ncycles         = 100 // number of learning cycles
	netsize         = 256 // number of colors used
	maxnetpos       = netsize - 1
	netbiasshift    = 4  // bias for colour values
	intbiasshift    = 16 // bias for fractions
	intbias         = 1 << intbiasshift
	gammashift      = 10
	gamma           = 1 << gammashift
	betashift       = 10
	beta            = intbias >> betashift // beta = 1/1024
	betagamma       = intbias << (gammashift - betashift)
	initrad         = netsize >> 3 // for 256 cols, radius starts
	radiusbiasshift = 6            // at 32.0 biased by 6 bits
	radiusbias      = 1 << radiusbiasshift
	initradius      = initrad * radiusbias // and decreases by a
	radiusdec       = 30                   // factor of 1/30 each cycle
	alphabiasshift  = 10                   // alpha starts at 1.0
	initalpha       = 1 << alphabiasshift
	radbiasshift    = 8
	radbias         = 1 << radbiasshift
	alpharadbshift  = alphabiasshift + radbiasshift
	alpharadbias    = 1 << alpharadbshift
	prime1          = 499
	prime2          = 491
	prime3          = 487
	prime4          = 503
	minpicturebytes = 3 * prime4
)

// NeuQuant is a neural network color quantizer
type NeuQuant struct {
	network   [][]int32 // [netsize][4] - the network itself
	netindex  []int32   // [256] - for network lookup - really 256
	bias      []int32   // [netsize] - bias array for learning
	freq      []int32   // [netsize] - freq array for learning
	radpower  []int32   // [initrad] - for radpower calculation
	pixels    []byte    // the input image in RGB format
	samplefac int       // sampling factor 1..30
}

// NewNeuQuant creates a new NeuQuant instance
// pixels: array of pixels in RGB format [r,g,b,r,g,b,...]
// samplefac: sampling factor 1 to 30 where lower is better quality
func NewNeuQuant(pixels []byte, samplefac int) *NeuQuant {
	return &NeuQuant{
		network:   make([][]int32, netsize),
		netindex:  make([]int32, 256),
		bias:      make([]int32, netsize),
		freq:      make([]int32, netsize),
		radpower:  make([]int32, initrad),
		pixels:    pixels,
		samplefac: samplefac,
	}
}

// init sets up arrays
func (nq *NeuQuant) init() {
	for i := 0; i < netsize; i++ {
		v := int32((i << (netbiasshift + 8)) / netsize)
		nq.network[i] = []int32{v, v, v, 0}
		nq.freq[i] = intbias / netsize
		nq.bias[i] = 0
	}
}

// BuildColormap builds the color map
// 1. initializes network
// 2. trains it
// 3. removes misconceptions
// 4. builds colorindex
func (nq *NeuQuant) BuildColormap() {
	nq.init()
	nq.learn()

	// gc
	nq.pixels = nil

	nq.unbiasnet()
	nq.inxbuild()
}

// GetColormap returns the color map as byte array [r,g,b,r,g,b,...]
func (nq *NeuQuant) GetColormap() []byte {
	colormap := make([]byte, netsize*3)
	index := make([]int, netsize)

	for i := 0; i < netsize; i++ {
		index[nq.network[i][3]] = i
	}

	k := 0
	for i := 0; i < netsize; i++ {
		j := index[i]
		colormap[k] = byte(nq.network[j][0])
		k++
		colormap[k] = byte(nq.network[j][1])
		k++
		colormap[k] = byte(nq.network[j][2])
		k++
	}
	return colormap
}

// LookupRGB looks for the closest r, g, b color in the map and returns its index
func (nq *NeuQuant) LookupRGB(r, g, b byte) int {
	// 注意：虽然 inxsearch 的参数名是 (b, g, r)，但实际期望的是 RGB 顺序
	// 这是原始代码的命名混淆，不要被参数名误导
	return nq.inxsearch(int32(r), int32(g), int32(b))
}

// unbiasnet unbiases network to give byte values 0..255 and record position i to prepare for sort
func (nq *NeuQuant) unbiasnet() {
	for i := 0; i < netsize; i++ {
		nq.network[i][0] >>= netbiasshift
		nq.network[i][1] >>= netbiasshift
		nq.network[i][2] >>= netbiasshift
		nq.network[i][3] = int32(i) // record color number
	}
}

// altersingle moves neuron i towards biased (b,g,r) by factor alpha
func (nq *NeuQuant) altersingle(alpha, i int32, b, g, r int32) {
	nq.network[i][0] -= (alpha * (nq.network[i][0] - b)) / initalpha
	nq.network[i][1] -= (alpha * (nq.network[i][1] - g)) / initalpha
	nq.network[i][2] -= (alpha * (nq.network[i][2] - r)) / initalpha
}

// alterneigh moves neurons in radius around index i towards biased (b,g,r) by factor alpha
func (nq *NeuQuant) alterneigh(radius int, i int, b, g, r int32) {
	lo := abs32(i - radius)
	hi := min(i+radius, netsize)

	j := i + 1
	k := i - 1
	m := 1

	for j < hi || k > lo {
		a := nq.radpower[m]
		m++

		if j < hi {
			p := nq.network[j]
			p[0] -= (a * (p[0] - b)) / alpharadbias
			p[1] -= (a * (p[1] - g)) / alpharadbias
			p[2] -= (a * (p[2] - r)) / alpharadbias
			j++
		}

		if k > lo {
			p := nq.network[k]
			p[0] -= (a * (p[0] - b)) / alpharadbias
			p[1] -= (a * (p[1] - g)) / alpharadbias
			p[2] -= (a * (p[2] - r)) / alpharadbias
			k--
		}
	}
}

// contest searches for biased BGR values
// finds closest neuron (min dist) and updates freq
// finds best neuron (min dist-bias) and returns position
func (nq *NeuQuant) contest(b, g, r int32) int {
	bestd := int32(0x7FFFFFFF) // math.MaxInt32 = 2147483647
	bestbiasd := bestd
	bestpos := -1
	bestbiaspos := bestpos

	for i := 0; i < netsize; i++ {
		n := nq.network[i]
		dist := abs32int(n[0]-b) + abs32int(n[1]-g) + abs32int(n[2]-r)

		if dist < bestd {
			bestd = dist
			bestpos = i
		}

		biasdist := dist - ((nq.bias[i]) >> (intbiasshift - netbiasshift))
		if biasdist < bestbiasd {
			bestbiasd = biasdist
			bestbiaspos = i
		}

		betafreq := nq.freq[i] >> betashift
		nq.freq[i] -= betafreq
		nq.bias[i] += betafreq << gammashift
	}

	nq.freq[bestpos] += beta
	nq.bias[bestpos] -= betagamma

	return bestbiaspos
}

// learn is the main learning loop
func (nq *NeuQuant) learn() {
	lengthcount := len(nq.pixels)
	alphadec := int32(30 + ((nq.samplefac - 1) / 3))
	samplepixels := lengthcount / (3 * nq.samplefac)
	delta := samplepixels / ncycles
	if delta == 0 {
		delta = 1
	}

	alpha := int32(initalpha)
	radius := int32(initradius)

	rad := int(radius >> radiusbiasshift)
	if rad <= 1 {
		rad = 0
	}

	for i := 0; i < rad; i++ {
		nq.radpower[i] = alpha * ((int32(rad*rad-i*i) * radbias) / int32(rad*rad))
	}

	var step int
	if lengthcount < minpicturebytes {
		nq.samplefac = 1
		step = 3
	} else if lengthcount%prime1 != 0 {
		step = 3 * prime1
	} else if lengthcount%prime2 != 0 {
		step = 3 * prime2
	} else if lengthcount%prime3 != 0 {
		step = 3 * prime3
	} else {
		step = 3 * prime4
	}

	pix := 0
	i := 0

	for i < samplepixels {
		b := (int32(nq.pixels[pix]) & 0xff) << netbiasshift
		g := (int32(nq.pixels[pix+1]) & 0xff) << netbiasshift
		r := (int32(nq.pixels[pix+2]) & 0xff) << netbiasshift

		j := nq.contest(b, g, r)

		nq.altersingle(alpha, int32(j), b, g, r)
		if rad != 0 {
			nq.alterneigh(rad, j, b, g, r)
		}

		pix += step
		if pix >= lengthcount {
			pix -= lengthcount
		}

		i++

		if i%delta == 0 {
			alpha -= alpha / alphadec
			radius -= radius / radiusdec
			rad = int(radius >> radiusbiasshift)

			if rad <= 1 {
				rad = 0
			}
			for j := 0; j < rad; j++ {
				nq.radpower[j] = alpha * ((int32(rad*rad-j*j) * radbias) / int32(rad*rad))
			}
		}
	}
}

// inxbuild sorts network and builds netindex[0..255]
func (nq *NeuQuant) inxbuild() {
	previouscol := int32(0)
	startpos := 0

	for i := 0; i < netsize; i++ {
		p := nq.network[i]
		smallpos := i
		smallval := p[1] // index on g

		// find smallest in i..netsize-1
		for j := i + 1; j < netsize; j++ {
			q := nq.network[j]
			if q[1] < smallval { // index on g
				smallpos = j
				smallval = q[1]
			}
		}

		// q := nq.network[smallpos]

		// swap p (i) and q (smallpos) entries
		if i != smallpos {
			nq.network[i], nq.network[smallpos] = nq.network[smallpos], nq.network[i]
			p = nq.network[i]
		}

		// smallval entry is now in position i
		if smallval != previouscol {
			nq.netindex[previouscol] = int32((startpos + i) >> 1)
			for j := previouscol + 1; j < smallval; j++ {
				nq.netindex[j] = int32(i)
			}
			previouscol = smallval
			startpos = i
		}
	}

	nq.netindex[previouscol] = int32((startpos + maxnetpos) >> 1)
	for j := previouscol + 1; j < 256; j++ {
		nq.netindex[j] = maxnetpos
	}
}

// inxsearch searches for BGR values 0..255 and returns a color index
func (nq *NeuQuant) inxsearch(b, g, r int32) int {
	bestd := int32(1000) // biggest possible dist is 256*3
	best := -1

	i := int(nq.netindex[g]) // index on g
	j := i - 1               // start at netindex[g] and work outwards

	for i < netsize || j >= 0 {
		if i < netsize {
			p := nq.network[i]
			dist := p[1] - g // inx key

			if dist >= bestd {
				i = netsize // stop iter
			} else {
				i++
				if dist < 0 {
					dist = -dist
				}
				a := p[0] - b
				if a < 0 {
					a = -a
				}
				dist += a

				if dist < bestd {
					a = p[2] - r
					if a < 0 {
						a = -a
					}
					dist += a

					if dist < bestd {
						bestd = dist
						best = int(p[3])
					}
				}
			}
		}

		if j >= 0 {
			p := nq.network[j]
			dist := g - p[1] // inx key - reverse dif

			if dist >= bestd {
				j = -1 // stop iter
			} else {
				j--
				if dist < 0 {
					dist = -dist
				}
				a := p[0] - b
				if a < 0 {
					a = -a
				}
				dist += a

				if dist < bestd {
					a = p[2] - r
					if a < 0 {
						a = -a
					}
					dist += a

					if dist < bestd {
						bestd = dist
						best = int(p[3])
					}
				}
			}
		}
	}

	return best
}

// Helper functions
func abs32(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func abs32int(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
