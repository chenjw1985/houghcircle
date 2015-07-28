package main

import (
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"runtime"
)

var (
	Gaos   []float64 //高斯模糊权重
	path   string    //当前路径
	cstep  int       //圆心误差
	radius float64   //圆直径
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	Gaos = []float64{0.0947416, 0.118318, 0.0947416, 0.118318, 0.147761, 0.118318, 0.0947416, 0.118318, 0.0947416}
	radius = 45.0
	cstep = 5
	path, _ = os.Getwd()
}

func main() {
	openImg("cache/test.jpg", i)
}

/**
 * 查找图片
 *
 **/
func openImg(filename string, page int) {
	reader, err := os.Open(filename)
	if err != nil {
		LOGS.Error(err.Error())
	}
	defer reader.Close()
	//reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
	m, _, err := image.Decode(reader)
	if err != nil {
		LOGS.Error(err.Error())
	}

	w, h := m.Bounds().Max.X, m.Bounds().Max.Y

	process(m, page, w, h, 2, cmdtype)
}

/**
 * 查找开始
 *, radius float64
 **/
func process(m image.Image, page, w, h, style int, normal bool) {
	//获取灰度值
	grayBox := getGrayColors(m, w, h, 2)
	//fmt.Println("grayBox end..........")

	//twoBox := getTwoColors(grayBox, w, h)

	gaosBox := getGaosColors(grayBox, w, h)

	twoBox := getTwoColors(gaosBox, w, h)

	var newBox []uint8
	var rw int
	var rh int

	if normal == true {
		newBox, rw, rh = clearPix(twoBox, w, h)
	} else {
		newBox, rw, rh = twoBox, w, h
	}

	pagescale = float64(rw) / A4[0]
	radius = radius * pagescale

	//霍夫换算
	hrect, max := HoughCircle(newBox, rw, rh, radius)
	if debug == true {
		fmt.Println("process", page, max)
	}
	if max > 250 {
		//绘制找出的公章位置
		img, c := findCircle(hrect, 3, rw, rh, radius)
		saveImg(img, "4-stamp--"+FILENAME)

		l := float64(1024) / float64(rw) * float64(c[1])
		whsize := float64(c[2]) / float64(c[1])
		t := float64(whsize) * float64(l)
		r := radius / float64(rw) * float64(l)

		if debug == true {
			fmt.Println(l, t, r, whsize)
			fmt.Println(page, int(l-r-35), int(t-r-50), int(r+10))
		}
		//转换为True公章n string,p,x,y,r int
		//STAMP_ARR = append(STAMP_ARR, createTrueStamp("中威科技", 1, 576, 774, 77))
		STAMP_ARR = append(STAMP_ARR, createTrueStamp(stampname, page, int(l-r-35), int(t-r-50), int(r+10)))
	}
}

/**
 * 灰度处理
 *
 **/
func getGrayColors(img image.Image, w, h, style int) []uint8 {
	var grayColor float64 = 0.0
	rect := make([]uint8, 0, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if style == 0 {
				grayColor = float64((r + g + b) / 3)
			} else if style == 1 {
				//grayColor = 0.11*float64(r) + 0.59*float64(g) + 0.3*float64(b)
				grayColor = 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b) + 0.5
			} else {
				grayColor = math.Min(float64(r), math.Min(float64(g), float64(b)))
			}
			rect = append(rect, uint8(grayColor))
		}
	}
	return rect
}

/**
 * 二值化处理
 *
 */
func getTwoColors(rect []uint8, w, h int) []uint8 {
	var AverageColor uint8
	twoColors := make([]uint8, 0, w*h)
	cache := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			AverageColor = getAverageColor(rect, w, h, x, y)
			if rect[x+(y*w)] < AverageColor {
				twoColors = append(twoColors, 255)
				cache.Set(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				twoColors = append(twoColors, 0)
				cache.Set(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	saveImg(cache, "2-Two--"+FILENAME)

	return twoColors
}

/**
 * 对二值化的图片进行高斯模糊
 *
 */
func getGaosColors(rect []uint8, w, h int) []uint8 {
	var GaosColor uint8
	GaosColors := make([]uint8, 0, w*h)
	cache := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			GaosColor = getGaosColor(rect, w, h, x, y)
			GaosColors = append(GaosColors, GaosColor)
			cache.Set(x, y, color.RGBA{GaosColor, GaosColor, GaosColor, 255})
		}
	}
	saveImg(cache, "1-gaos--"+FILENAME)

	return GaosColors
}

/**
 * 获取四周9个像素值
 */
func getPoint9Color(rect []uint8, w, h, x, y int) []uint8 {
	/**
	周边点矩阵
	 _________________________________________
	|									      |
	| r1(x-1,y-1)| r2(x,y-1)  | r3(x+1,y-1)   |
	|_________________________________________|
	|		     |		      |				  |
	| r4(x-1,y)  | r5(x,y)    | r6(x+1,y)	  |
	|_________________________________________|
	|		     |		      |				  |
	| r7(x-1,y+1)| r8(x,y+1)  | r9(x+1,y+1)	  |
	|_________________________________________|

	**/

	var r1 uint8 = 0
	if x == 0 || y == 0 {
		r1 = 255
	} else {
		r1 = rect[x-1+(y-1)*w]
	}

	var r2 uint8 = 0
	if y == 0 {
		r2 = 255
	} else {
		r2 = rect[x+(y-1)*w]
	}

	var r3 uint8 = 0
	if x == w-1 || y == 0 {
		r3 = 255
	} else {
		r3 = rect[x+1+(y-1)*w]
	}

	var r4 uint8 = 0
	if x == 0 {
		r4 = 255
	} else {
		r4 = rect[x-1+(y*w)]
	}

	var r5 uint8 = rect[x+(y*w)]

	var r6 uint8 = 0
	if x == w-1 {
		r6 = 255
	} else {
		r6 = rect[x+1+(y*w)]
	}

	var r7 uint8 = 0
	if x == 0 || y == h-1 {
		r7 = 255
	} else {
		r7 = rect[x-1+(y+1)*w]
	}

	var r8 uint8 = 0
	if y == h-1 {
		r8 = 255
	} else {
		r8 = rect[x+(y+1)*w]
	}

	var r9 uint8 = 0
	if x == w-1 || y == h-1 {
		r9 = 255
	} else {
		r9 = rect[x+1+(y+1)*w]
	}

	return []uint8{r1, r2, r3, r4, r5, r6, r7, r8, r9}
}

/**
 * 根据四周8个像素判断中间隔值
 */
func getAverageColor(rect []uint8, w, h, x, y int) uint8 {
	point9 := getPoint9Color(rect, w, h, x, y)
	return uint8((point9[0]+point9[1]+point9[2]+point9[3]+point9[4]+point9[5]+point9[6]+point9[7]+point9[8])/9 + 128)
}

/**
 * 高斯模糊算法
 **/
func getGaosColor(rect []uint8, w, h, x, y int) uint8 {
	point9 := getPoint9Color(rect, w, h, x, y)
	i, l := 0, 9
	var point float64
	for ; i < l; i++ {
		point += float64(point9[i]) * Gaos[i]
	}
	return uint8(point)
}

/**
 * 霍夫变换
 **/
func HoughCircle(rect []uint8, w, h int, radius float64) ([]int, int) {

	var t float64
	x0, y0 := 0, 0
	//初始化极坐标
	acc := make([]int, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc = append(acc, 0)
		}
	}
	//绘制极坐标
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			if (rect[x+(y*w)] & 0xff) == 255 {
				for theta := 0; theta < 360; theta++ {
					t = (float64(theta) * 3.14159265) / 180
					x0 = int(math.Floor(float64(x) - radius*math.Cos(t)))
					y0 = int(math.Floor(float64(y) - radius*math.Sin(t)))
					if x0 < w && x0 > 0 && y0 < h && y0 > 0 {
						acc[x0+(y0*w)] += 1
					}
				}
			}
		}
	}
	max := 0

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			if acc[x+(y*w)] > max {
				max = acc[x+(y*w)]
			}
		}
	}
	//根据最大值，实现极坐标空间的灰度值归一处理
	v := 0
	cache := image.NewNRGBA(image.Rect(0, 0, w, h))

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			v = int((float64(acc[x+(y*w)]) / float64(max)) * 255.0)
			acc[x+(y*w)] = 0 | (v<<16 | v<<8 | v)
			cache.Set(x, y, color.RGBA{uint8(acc[x+(y*w)]), uint8(acc[x+(y*w)]), uint8(acc[x+(y*w)]), 255})
		}
	}
	saveImg(cache, "3-hough--"+FILENAME)

	return acc, max
}

/**
 * 查找圆
 *
 **/
func findCircle(acc []int, accsize, w, h int, radius float64) (draw.Image, []int) {
	/**
	[0,1,2,3,4,5,6,7,8]
	**/
	results := make([]int, accsize*3, accsize*3)
	output := image.NewNRGBA(image.Rect(0, 0, w, h))

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {

			v := acc[x+(y*w)] & 0xff
			//if its higher then lowest value add it and then sort
			if v > results[(accsize-1)*3]-10 {
				//add to bottom of array
				results[(accsize-1)*3] = v
				results[(accsize-1)*3+1] = x
				results[(accsize-1)*3+2] = y
				//shift up until its in right place
				i := (accsize - 2) * 3
				for i >= 0 && results[i+3] > results[i] {
					for j := 0; j < 3; j++ {
						temp := results[i+3+j]
						results[i+j] = results[i+3+j]
						results[i+3+j] = temp
					}
					i = i - 3
					if i < 0 {
						break
					}
				}
			}

		}
	}
	center := make([]int, accsize*3, accsize*3)
	// 根据找到的半径R，中心点像素坐标p(x, y)，绘制圆在原图像上
	for i := accsize - 1; i >= 0; i-- {
		drawCircle(output, results[i*3], results[i*3+1], results[i*3+2], radius)
		center[i*3] = results[i*3]
		center[i*3+1] = results[i*3+1]
		center[i*3+2] = results[i*3+2]
	}
	return output, center
}

/**
 * 绘制圆
 *
 **/
func drawCircle(output *image.NRGBA, pix, xc, yc int, radius float64) {
	//圆心
	//fmt.Println(xc, yc)
	pix = 255
	x, y := 0, 0
	r2 := radius * radius
	//绘制圆的四个方向的定点
	output.Set(xc, yc+int(radius), color.RGBA{255, 0, 0, 255})
	output.Set(xc, yc-int(radius), color.RGBA{255, 0, 0, 255})
	output.Set(xc+int(radius), yc, color.RGBA{255, 0, 0, 255})
	output.Set(xc-int(radius), yc, color.RGBA{255, 0, 0, 255})

	//y = int(radius) ???
	x = 1
	y = int(math.Sqrt(r2-1) + 0.5)

	// 边缘填充算法， 其实可以直接对循环所有像素，计算到做中心点距离来做
	for x < y {
		output.Set(xc+x, yc+y, color.RGBA{255, 0, 0, 255})
		output.Set(xc+x, yc-y, color.RGBA{255, 0, 0, 255})
		output.Set(xc-x, yc+y, color.RGBA{255, 0, 0, 255})
		output.Set(xc-x, yc-y, color.RGBA{255, 0, 0, 255})
		output.Set(xc+y, yc+x, color.RGBA{255, 0, 0, 255})
		output.Set(xc+y, yc-x, color.RGBA{255, 0, 0, 255})
		output.Set(xc-y, yc+x, color.RGBA{255, 0, 0, 255})
		output.Set(xc-y, yc-x, color.RGBA{255, 0, 0, 255})
		x += 1
		y = (int)(math.Sqrt(r2-float64(x)*float64(x)) + 0.5)
	}
	if x == y {
		output.Set(xc+x, yc+y, color.RGBA{255, 0, 0, 255})
		output.Set(xc+x, yc-y, color.RGBA{255, 0, 0, 255})
		output.Set(xc-x, yc+y, color.RGBA{255, 0, 0, 255})
		output.Set(xc-x, yc-y, color.RGBA{255, 0, 0, 255})
	}
}

/**
 * 生成图片
 *
 **/
func saveImg(img draw.Image, name string) error {
	if debug == false {
		return nil
	}
	draw.Draw(img, img.Bounds(), img, image.ZP, draw.Src)
	f, err := os.Create("temps/" + name + ".jpeg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = jpeg.Encode(f, img, &jpeg.Options{90})
	//err = png.Encode(f3, __m)
	if err != nil {
		panic(err)
	}
	return err
}

func initToByte(i int) []byte {
	var l, h, h1, h2 uint8 = uint8(i >> 24), uint8(i >> 16), uint8(i >> 8), uint8(i & 0xff)
	nbyte := []byte{h2, h1, h, l}
	return nbyte
}

func byteToInt(b []byte) int {
	v0 := int(b[3]) << 24
	v1 := int(b[2]) << 16
	v2 := int(b[1]) << 8
	v3 := int(b[0]) & 0xff
	return int(v0) + int(v1) + int(v2) + int(v3)
}

func byteToHex(b []byte) string {
	hexStr := hex.EncodeToString(b)
	if len(hexStr) < 8 {
		for 8-len(hexStr) > 0 {
			hexStr = "0" + hexStr
		}
	}
	return hexStr
}

// Get bytes to file.
// if non-exist, create this file.
func File_get_contents(filename string) ([]byte, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return []byte(""), err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return []byte(""), err
	}
	data := make([]byte, stat.Size())
	result, err := f.Read(data)
	if int64(result) == stat.Size() {
		return data, err
	}
	return []byte(""), err
}

// Put bytes to file.
// if non-exist, create this file.
func File_put_contents(filename string, content []byte) error {
	fp, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer fp.Close()
	_, err = fp.Write(content)
	return err
}

/**
毫米转像素
mm 毫米单位
dpi 分辨率
**/
func mm2px(mm, dpi float64) float64 {
	return mm / 10 * dpi * 0.3937
}

/**
像素转毫米
px 像素单位
dpi 分辨率
**/
func px2mm(px, dpi float64) float64 {
	return px / dpi * 2.54 * 10
}