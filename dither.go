package gifencoder

// DitheringKernel 定义抖动核心
type DitheringKernel [][]float64

// 预定义的抖动核心
var (
	// FalseFloydSteinberg 抖动核心
	FalseFloydSteinberg = DitheringKernel{
		{3.0 / 8.0, 1, 0},
		{3.0 / 8.0, 0, 1},
		{2.0 / 8.0, 1, 1},
	}

	// FloydSteinberg 抖动核心（最常用）
	FloydSteinberg = DitheringKernel{
		{7.0 / 16.0, 1, 0},
		{3.0 / 16.0, -1, 1},
		{5.0 / 16.0, 0, 1},
		{1.0 / 16.0, 1, 1},
	}

	// Stucki 抖动核心
	Stucki = DitheringKernel{
		{8.0 / 42.0, 1, 0},
		{4.0 / 42.0, 2, 0},
		{2.0 / 42.0, -2, 1},
		{4.0 / 42.0, -1, 1},
		{8.0 / 42.0, 0, 1},
		{4.0 / 42.0, 1, 1},
		{2.0 / 42.0, 2, 1},
		{1.0 / 42.0, -2, 2},
		{2.0 / 42.0, -1, 2},
		{4.0 / 42.0, 0, 2},
		{2.0 / 42.0, 1, 2},
		{1.0 / 42.0, 2, 2},
	}

	// Atkinson 抖动核心
	Atkinson = DitheringKernel{
		{1.0 / 8.0, 1, 0},
		{1.0 / 8.0, 2, 0},
		{1.0 / 8.0, -1, 1},
		{1.0 / 8.0, 0, 1},
		{1.0 / 8.0, 1, 1},
		{1.0 / 8.0, 0, 2},
	}
)

// DitherMethod 抖动方法
type DitherMethod string

const (
	DitherNone                DitherMethod = "none"
	DitherFloydSteinberg      DitherMethod = "FloydSteinberg"
	DitherFalseFloydSteinberg DitherMethod = "FalseFloydSteinberg"
	DitherStucki              DitherMethod = "Stucki"
	DitherAtkinson            DitherMethod = "Atkinson"
)

// ditherPixels 对像素应用抖动算法
// method: 抖动方法名称
// serpentine: 是否使用蛇形扫描
func (ge *GIFEncoder) ditherPixels(method DitherMethod, serpentine bool) {
	// 选择抖动核心
	var kernel DitheringKernel
	switch method {
	case DitherFalseFloydSteinberg:
		kernel = FalseFloydSteinberg
	case DitherFloydSteinberg:
		kernel = FloydSteinberg
	case DitherStucki:
		kernel = Stucki
	case DitherAtkinson:
		kernel = Atkinson
	default:
		// 未知的抖动方法，回退到不抖动
		ge.indexPixels()
		return
	}

	width := ge.width
	height := ge.height
	data := ge.pixels
	direction := 1
	if serpentine {
		direction = -1
	}

	ge.indexedPixels = make([]byte, len(ge.pixels)/3)

	for y := 0; y < height; y++ {
		// 蛇形扫描：每行改变方向
		if serpentine {
			direction = -direction
		}

		// 设置行的起始和结束位置
		var x, xEnd int
		if direction == 1 {
			x = 0
			xEnd = width
		} else {
			x = width - 1
			xEnd = -1
		}

		// 扫描当前行
		for x != xEnd {
			index := y*width + x

			// 获取原始颜色
			idx := index * 3
			r1 := int(data[idx])
			g1 := int(data[idx+1])
			b1 := int(data[idx+2])

			// 找到最接近的调色板颜色
			colorIdx := ge.findClosestRGB(byte(r1), byte(g1), byte(b1))
			ge.usedEntry[colorIdx] = true
			ge.indexedPixels[index] = byte(colorIdx)

			// 获取量化后的颜色
			paletteIdx := colorIdx * 3
			r2 := int(ge.colorTab[paletteIdx])
			g2 := int(ge.colorTab[paletteIdx+1])
			b2 := int(ge.colorTab[paletteIdx+2])

			// 计算量化误差
			er := r1 - r2
			eg := g1 - g2
			eb := b1 - b2

			// 将误差扩散到邻近像素
			var i, iEnd int
			if direction == 1 {
				i = 0
				iEnd = len(kernel)
			} else {
				i = len(kernel) - 1
				iEnd = -1
			}

			for i != iEnd {
				x1 := int(kernel[i][1])
				y1 := int(kernel[i][2])

				// 检查邻近像素是否在图像范围内
				nx := x + x1
				ny := y + y1
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					d := kernel[i][0]
					nIdx := (ny*width + nx) * 3

					// 扩散误差，确保值在 0-255 范围内
					data[nIdx] = clamp(int(data[nIdx]) + int(float64(er)*d))
					data[nIdx+1] = clamp(int(data[nIdx+1]) + int(float64(eg)*d))
					data[nIdx+2] = clamp(int(data[nIdx+2]) + int(float64(eb)*d))
				}

				if direction == 1 {
					i++
				} else {
					i--
				}
			}

			x += direction
		}
	}
}

// clamp 将值限制在 0-255 范围内
func clamp(value int) byte {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return byte(value)
}
