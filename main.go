package main

import (
	"strings"
	"os"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"flag"
	"fmt"
	"image/color/palette"
	"argulox.top/argulox/ImageSpliter.git/commons"
	textdraw "argulox.top/argulox/ImageSpliter.git/draw"
	"github.com/nfnt/resize"
	"image/jpeg"
	"argulox.top/argulox/ImageSpliter.git/models"
)

type TrajectoryMeta struct {
	Img       *image.Image
	Point     *image.Point
	Sentences models.Sentence
}

type FlyTrajectory struct {
	Width  int
	Direct int
	PicNum int
	Img    image.Image
}

type BloomTrajectory struct {
	PicNum int
	Width  int
	Img    image.Image
}

func (c *BloomTrajectory) calc() chan TrajectoryMeta {
	var ch = make(chan TrajectoryMeta)
	go func() {
		center := image.Point{X: c.Width / 2, Y: c.Width / 2}
		originW := c.Width
		var scale float32 = 1.0
		for i := 0; i < c.PicNum; i++ {
			if i%3 == 0 {
				scale = 1.0
			} else {
				scale -= 0.1
			}
			width := float32(originW) * scale
			p := image.Point{X: center.X - int(width/2), Y: center.Y - int(width/2)}
			img := resize.Resize(uint(width), 0, c.Img, resize.Lanczos3)
			ch <- TrajectoryMeta{Img: &img, Point: &p}
		}
	}()
	return ch
}

type Trajectory interface {
	calc() chan TrajectoryMeta
}

func (c *FlyTrajectory) calc() chan TrajectoryMeta {
	var ch = make(chan TrajectoryMeta)
	dir := commons.If(c.Direct >= 0, 1, -1).(int)
	go func() {
		for i := 0; i < c.PicNum; i++ {
			if i == 0 {
				p := image.Point{X: int(float32(dir) * float32(c.Width) / 2.0), Y: -100}
				sen := models.Sentence{Range: []int{i, i}, X: int(float32(c.Width)/2.0 - float32(dir)*float32(c.Width)/2.0), Y: 50}
				ch <- TrajectoryMeta{&c.Img, &p, sen}
				continue
			} else if i == c.PicNum-1 {
				p := image.Point{X: int(float32(dir) * -float32(c.Width) / 2.0), Y: -100}
				sen := models.Sentence{Range: []int{i, i}, X: int(float32(c.Width)/2.0 + float32(dir)*float32(c.Width)/2.0), Y: 50}
				ch <- TrajectoryMeta{&c.Img, &p, sen}
			} else {
				var speed float32 = 3.0
				p := image.Point{X: int(float32(dir) * -float32(i) * speed), Y: -100}
				sen := models.Sentence{Range: []int{i, i}, X: int(float32(c.Width)/2.0 - float32(dir)*float32(i)*speed), Y: 50}
				ch <- TrajectoryMeta{&c.Img, &p, sen}
			}
		}
	}()
	return ch
}

func readImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("Could not open file %s. Error: %s\n", path, err)
		return nil, err
	}
	defer f.Close()
	img, _ := jpeg.Decode(f)
	if err != nil {
		fmt.Printf("Decode failed %s. Error: %s\n", path, err)
		return nil, err
	}
	//固定大小成256*256
	img = resize.Resize(256, 0, img, resize.Lanczos3)
	return img, nil
}

func makeOnlyFly(anim *gif.GIF, paths []string, delay int) error {
	if len(paths) < 3 {
		panic("picture number must not less than 3")
	}
	dir := -1
	for _, path := range paths[0:3] {
		dir *= -1
		if err := makeFly(anim, path, 14, delay, dir); err != nil {
			return err
		}

	}
	return nil
}

func makeBling(anim *gif.GIF, paths []string, delay int) error {
	if len(paths) < 3 {
		panic("picture number must not less than 3")
	}
	for _, path := range paths[0:2] {
		if err := makeFly(anim, path, 7, delay, 1); err != nil {
			return err
		}

	}
	if err := makeBoom(anim, paths[2], 9, delay); err != nil {
		return err
	}
	return nil
}

var man = textdraw.NewDrawTextManager("./STHeiti Medium.ttc", 72, "none")

func makeFly(anim *gif.GIF, path string, picNum, delay int, direct int) error {

	img, err := readImage(path)
	if err != nil {
		return err
	}
	log.Print("load image ", path, " ok")

	flyTrajectory := FlyTrajectory{Width: img.Bounds().Dx(), Direct: direct, PicNum: picNum, Img: img}
	backgroundHeight := img.Bounds().Dy() + 100
	generator := flyTrajectory.calc()
	for i := 0; i < picNum; i++ {
		x := <-generator
		backgroundRect := image.Rect(0, 0, img.Bounds().Max.X, backgroundHeight)
		paletted := image.NewPaletted(backgroundRect, palette.Plan9)
		draw.Draw(paletted, backgroundRect, &image.Uniform{color.White}, image.ZP, draw.Src)

		rect := image.Rectangle{Min: image.ZP, Max: image.Point{X: img.Bounds().Dx(), Y: img.Bounds().Dy() + 100}}
		draw.FloydSteinberg.Draw(paletted, rect, img, *x.Point)

		paletted = man.DrawTextInner(paletted, x.Sentences.X, x.Sentences.Y, "测试文字", 32, color.Black)

		anim.Image = append(anim.Image, paletted)
		anim.Delay = append(anim.Delay, delay)
	}
	return nil
}
func makeBoom(anim *gif.GIF, path string, picNum, delay int) error {

	//最后一张以闪烁的方式显示
	if img, err := readImage(path); err != nil {
		return err
	} else {
		bloomTrajectory := BloomTrajectory{picNum, img.Bounds().Dx(), img}
		backgroundHeight := img.Bounds().Dy() + 100
		generator := bloomTrajectory.calc()
		for i := 0; i < 9; i++ {
			x := <-generator
			backgroundRect := image.Rect(0, 0, img.Bounds().Max.X, backgroundHeight)
			paletted := image.NewPaletted(backgroundRect, palette.Plan9)
			draw.Draw(paletted, backgroundRect, &image.Uniform{color.White}, image.ZP, draw.Src)

			drawRect := image.Rectangle{Min: *x.Point, Max: image.Pt((*x.Point).X+(*x.Img).Bounds().Dx(), (*x.Point).Y+(*x.Img).Bounds().Dy())}
			draw.FloydSteinberg.Draw(paletted, drawRect, *x.Img, image.ZP)
			fmt.Println((*x.Img).Bounds(), " ", *x.Point)

			anim.Image = append(anim.Image, paletted)
			anim.Delay = append(anim.Delay, delay)
		}
	}
	return nil
}

func main() {

	var p, output string
	var delay int

	flag.StringVar(&p, "p", "", "图片路径,多个图片逗号分隔")
	flag.StringVar(&output, "o", "output.gif", "生成gif的文件名")
	flag.IntVar(&delay, "d", 5, "每张图片的展示时间*15毫秒")
	flag.Parse()

	if p == "" {
		fmt.Println("请输入图片路径")
		flag.PrintDefaults()
		return
	}

	paths := strings.Split(p, ",")

	anim := gif.GIF{}
	if err := makeOnlyFly(&anim, paths, delay); err != nil {
		log.Fatal(err)
	}

	f, _ := os.Create(output)
	defer f.Close()
	gif.EncodeAll(f, &anim)
}
