package authservicelogic

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/big"
)

// 数字点阵直接绘制成 PNG，响应中不携带文字答案或可检索文本元数据。
var digitGlyphs = [10][7]string{
	{"01110", "11011", "11011", "11011", "11011", "11011", "01110"},
	{"00110", "01110", "00110", "00110", "00110", "00110", "01111"},
	{"01110", "11011", "00011", "00110", "01100", "11000", "11111"},
	{"11110", "00011", "00011", "01110", "00011", "00011", "11110"},
	{"00011", "00111", "01111", "11011", "11111", "00011", "00011"},
	{"11111", "11000", "11000", "11110", "00011", "11011", "01110"},
	{"01110", "11000", "11000", "11110", "11011", "11011", "01110"},
	{"11111", "00011", "00110", "00110", "01100", "01100", "01100"},
	{"01110", "11011", "11011", "01110", "11011", "11011", "01110"},
	{"01110", "11011", "11011", "01111", "00011", "00011", "01110"},
}

func captchaImage(code string) (string, error) {
	const width, height = 176, 64
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{244, 247, 255, 255}), image.Point{}, draw.Src)
	var noise [1024]byte
	if _, err := rand.Read(noise[:]); err != nil {
		return "", err
	}
	for i := 0; i < 240; i++ {
		img.Set(int(noise[i*3])%width, int(noise[i*3+1])%height, color.RGBA{noise[i*3+2]/2 + 70, 130, 180, 255})
	}
	for index, digit := range code {
		glyph := digitGlyphs[digit-'0']
		xOffset, yOffset := 12+index*40, 13+int(noise[730+index])%7
		tone := color.RGBA{35 + noise[740+index]%50, 55 + noise[750+index]%45, 120 + noise[760+index]%50, 255}
		phase := float64(noise[770+index]) / 25
		for y, row := range glyph {
			for x, bit := range row {
				if bit != '1' {
					continue
				}
				for py := 0; py < 5; py++ {
					for px := 0; px < 5; px++ {
						cy := y*5 + py
						cx := xOffset + x*5 + px + int(2.8*math.Sin(float64(cy)/9+phase))
						img.Set(cx, yOffset+cy, tone)
					}
				}
			}
		}
	}
	for i := 0; i < 2; i++ {
		for x := 0; x < width; x++ {
			y := 20 + i*22 + int(5*math.Sin(float64(x)/20+float64(noise[800+i])))
			img.Set(x, y, color.RGBA{140, 165, 210, 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func randomDigits(length int) (string, error) {
	value := make([]byte, length)
	for i := range value {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		value[i] = '0' + byte(n.Int64())
	}
	return string(value), nil
}
