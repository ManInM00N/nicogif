# NicoGIF Encoder

纯 Go 语言实现的 GIF 编码库，移植自 [gif.js](https://github.com/jnordberg/gif.js)。使用 NeuQuant 神经网络算法进行高质量颜色量化。

[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)


### 安装

```bash
go get github.com/ManInM00N/nicogif/gifencoder
```

### 基础使用

```go
package main

import (
    "image"
    "image/color"
    "os"
    "github.com/ManInM00N/nicogif/gifencoder"
)

func main() {
    // 创建图像帧
    frames := []image.Image{
        createFrame(0),
        createFrame(1),
        createFrame(2),
    }
    
    // 设置每帧延迟（毫秒）
    delays := []int{100, 100, 100}
    
    // 编码为 GIF
    gifData, _ := gifencoder.EncodeGIF(frames, delays)
    
    // 保存文件
    os.WriteFile("output.gif", gifData, 0644)
}

func createFrame(n int) image.Image {
    img := image.NewRGBA(image.Rect(0, 0, 100, 100))
    // 绘制图像...
    return img
}
```

### 示例 1: 手动控制编码器

```go
func advancedUsage() {
    encoder := gifencoder.NewGIFEncoder(300, 300)
    
    // 设置选项
    encoder.SetRepeat(0)        // 0 = 无限循环
    encoder.SetQuality(10)      // 1-30, 值越小质量越好
    encoder.SetFrameRate(10)    // 10 FPS
    
    // 可选：设置透明色
    encoder.SetTransparent(&color.RGBA{255, 255, 255, 255})
    
    // 添加帧
    for i := 0; i < 20; i++ {
        img := createFrame(300, 300, i)
        encoder.SetDelay(100) // 每帧 100ms
        encoder.AddFrame(img)
    }
    
    encoder.Finish()
    gifData := encoder.GetData()
    
    os.WriteFile("advanced.gif", gifData, 0644)
}
```

### 示例 2: 使用自定义选项

```go
func withOptions() {
    frames := []image.Image{ /* ... */ }
    
    opts := gifencoder.EncodeOptions{
        Width:   200,
        Height:  200,
        Repeat:  0,     // 无限循环
        Quality: 5,     // 高质量
        Dither:  false,
        Delays:  []int{80, 80, 80, 80, 80},
    }
    
    gifData, err := gifencoder.EncodeGIFWithOptions(frames, opts)
    if err != nil {
        panic(err)
    }
    
    os.WriteFile("custom.gif", gifData, 0644)
}
```


### 创建编码器

```go
func NewGIFEncoder(width, height int) *GIFEncoder
```

### 配置方法

| 方法 | 说明 |
|------|------|
| `SetDelay(ms int)` | 设置帧延迟（毫秒） |
| `SetFrameRate(fps int)` | 设置帧率 |
| `SetRepeat(repeat int)` | 设置重复次数（-1=播放一次, 0=无限循环） |
| `SetQuality(quality int)` | 设置质量（1-30，越小越好） |
| `SetTransparent(*color.RGBA)` | 设置透明色 |
| `SetDither(bool)` | 启用/禁用抖动 |
| `SetGlobalPalette([]byte)` | 设置全局调色板 |
| `SetDispose(int)` | 设置帧处理方式 |

### 编码方法

| 方法 | 说明 |
|------|------|
| `AddFrame(image.Image) error` | 添加一帧 |
| `Finish()` | 完成编码 |
| `GetData() []byte` | 获取 GIF 数据 |
| `Stream() *ByteArray` | 获取输出流 |

### 便捷函数

```go
// 快速编码
func EncodeGIF(images []image.Image, delays []int) ([]byte, error)

// 带选项编码
type EncodeOptions struct {
    Width         int
    Height        int
    Repeat        int
    Quality       int
    Dither        bool
    GlobalPalette []byte
    Delays        []int
}

func EncodeGIFWithOptions(images []image.Image, opts EncodeOptions) ([]byte, error)
```

## ⚙️ 性能优化建议

### 质量参数

- **1-5**: 最佳质量，编码较慢（适合重要图像）
- **10-15**: 推荐值，质量和速度平衡（日常使用）
- **20-30**: 快速编码，质量略降（快速预览）

```go
encoder.SetQuality(10) // 推荐值
```

### 全局调色板

对于颜色相似的多帧动画，使用全局调色板可显著减小文件大小：

```go
encoder.SetGlobalPalette(nil) // 从第一帧自动生成
```

### 图像尺寸

较小的图像尺寸会显著提高编码速度：

```go
// 200x200 比 800x800 快约 16 倍
encoder := gifencoder.NewGIFEncoder(200, 200)
```

## ❓ 常见问题

### Q: 如何减小 GIF 文件大小？

**A:** 几种方法：
1. 使用全局调色板：`encoder.SetGlobalPalette(nil)`
2. 提高质量参数：`encoder.SetQuality(15-20)`
3. 减小图像尺寸
4. 减少帧数或增加帧延迟

### Q: 为什么颜色不准确？

**A:** GIF 最多只支持 256 色。对于颜色丰富的图像，降低质量参数（1-5）可获得更好的颜色匹配：

```go
encoder.SetQuality(1) // 最佳质量，但较慢
```

### Q: 如何创建透明背景 GIF？

**A:**

```go
encoder.SetTransparent(&color.RGBA{255, 255, 255, 255})
```

### Q: 支持哪些图像格式作为输入？

**A:** 支持所有实现了 `image.Image` 接口的类型，包括：
- `image.RGBA`
- `image.NRGBA`
- `image.Gray`
- PNG、JPEG 解码后的图像等

## 🙏 致谢

- [gif.js](https://github.com/jnordberg/gif.js) - 原始 JavaScript 实现
- Anthony Dekker - NeuQuant 算法作者
- Lempel, Ziv, Welch - LZW 压缩算法

## 🔗 相关链接

- [GIF89a 规范](https://www.w3.org/Graphics/GIF/spec-gif89a.txt)
- [NeuQuant 算法](http://members.ozemail.com.au/~dekker/NEUQUANT.HTML)
- [gif.js GitHub](https://github.com/jnordberg/gif.js)

## 🐛 问题反馈

如果遇到问题，请提交 Issue 到 GitHub。
