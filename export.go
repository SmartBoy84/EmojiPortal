package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

func Export(fileName string, img image.Image, quality int) (err error) {

	if quality > 100 {
		return fmt.Errorf("quality can only be (0,100]")
	}

	ext := filepath.Ext(fileName)
	if ext == "" {
		if quality == 100 {
			ext = ".png"
		} else {
			ext = ".jpg"
		}
		fileName = fmt.Sprintf("%s%s", fileName, ext)

	} else if ext != ".png" && ext != ".jpg" {
		return fmt.Errorf("only support jpg/png formats for output but got [%s]", ext)
	
		} else if quality < 100 && ext == ".jpg" {
		fmt.Printf("[warning] quality value specified with png as the output format so ignored")
		quality = 100
	}

	out, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	fmt.Printf("Exporting to %s\n", fileName)

	switch ext {
	case ".jpg":
		err = jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
	case ".png":
		err = png.Encode(out, img)
	}

	return err
}

func (emojis EmojiKeg) Chunky(folderName string) error { // depecrated, only use cartridges

	for _, brand := range emojis {
		if err := brand.ExportEmojis(fmt.Sprintf("%s/%s", folderName, brand.name)); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) ExportEmojis(folderName string) error {

	fmt.Printf("\nExporting emojis for %s", brand.name)

	var scalar image.Rectangle
	var err error

	if scalar, err = brand.GetScalar(1); err != nil {
		return err
	}

	if err := os.MkdirAll(folderName, 0700); err != nil {
		return err
	}

	for i, emoji := range brand.emojis.list {
		img := Resize(emoji.img, scalar)

		if err := Export(fmt.Sprintf("%s/%d__%s", folderName, i, emoji.name), img, 100); err != nil {
			return err
		}
	}

	return nil
}

func (emojis EmojiKeg) Export(folderName string) error {
	for _, brand := range emojis {
		if err := brand.CreateCartridge(fmt.Sprintf("%s/%s", folderName, brand.name)); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) CreateCartridge(fileName string) error {

	fmt.Printf("Saving cartridge %s -> %s\n", brand.name, fileName)

	var err error
	var scalar image.Rectangle

	if scalar, err = brand.GetScalar(1); err != nil {
		return err
	}

	if err := os.MkdirAll(path.Dir(fileName), 0700); err != nil {
		return err
	}

	/* using our method above we get a number such that x^2 = number of emojis
	=> x * x = (y + z) * (y + z) where y is a whole number and z is %1
	=> Now we want k such that y * (y + z + k) = (y+z)^2
	y^2 +yz + yk = y^2 + 2yz + z^2
	=> yz + z^2 = yk
	so, => k = (yz + z^2)/y
	where we know y to be the floor of the square, z to be %1 of the square */

	square := math.Sqrt(float64(len(brand.emojis.list)))
	whole := math.Trunc(square) // y
	tail := square - whole      // z

	canvasSize := image.Point{
		scalar.Dx() * int(whole),
		scalar.Dy() * int(math.Ceil(((whole*tail)+math.Pow(whole, 2))/whole)+1), // simply the formula applied (idk why +1 row is needed)
	}

	canvas := GetTransparent(color.RGBA{}, image.Rectangle{
		image.Point{0, 0},
		canvasSize,
	})

	currentPosition := image.Rectangle{image.Point{0, 0}, scalar.Size()}

	// translators
	shiftRight := image.Point{scalar.Dx(), 0}
	shiftDown := image.Point{0, scalar.Dy()}

	for _, emoji := range brand.emojis.list {

		if emoji == nil {
			continue
		}

		scaledEmoji := Resize(emoji.img, scalar)

		draw.Draw(canvas, currentPosition, scaledEmoji, image.Point{0, 0}, draw.Over)

		if currentPosition.Max.X == canvasSize.X {
			currentPosition = currentPosition.Add(shiftDown)

			currentPosition.Min.X = 0
			currentPosition.Max.X = scalar.Dx()

		} else {
			currentPosition = currentPosition.Add(shiftRight)
		}
	}

	if err := Export(fmt.Sprintf("%s-%dx%d.png", fileName, scalar.Dx(), scalar.Dy()), canvas, 100); err != nil {
		return err
	}
	return nil
}

func (emojis EmojiKeg) Emojify(inputName string, outputPath string, imageScale float64, quality int) error {

	for _, brand := range emojis {
		if err := brand.Emojify(inputName, fmt.Sprintf("%s/%s", outputPath, brand.name), imageScale, quality); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) Emojify(inputName string, outputName string, imageScale float64, quality int) error {

	if len(outputName) == 0 {
		name := filepath.Base(inputName)
		outputName = fmt.Sprintf("%s-%s", strings.TrimSuffix(name, filepath.Ext(name)), brand.name)
	}

	fmt.Printf("Emojifying %s with brand %s\n", inputName, brand.name)

	imageData, err := OpenImage(inputName)
	if err != nil {
		return err
	}

	img, err := brand.ConvertImage(imageData, imageScale)
	if err != nil {
		return err
	}

	if err := Export(outputName, img, quality); err != nil {
		return err
	}

	return nil
}

func (brand *Brand) ConvertImage(img image.Image, imageScale float64) (image.Image, error) {

	imageScalar, err := CreateScalar(img, imageScale)
	if err != nil {
		return nil, err
	}

	if len(brand.emojis.list) == 0 {
		return nil, fmt.Errorf("no emojis found")
	}

	emojiScalar := brand.emojis.list[0].img.Bounds()

	canvasSize := image.Rectangle{Max: image.Point{
		X: imageScalar.Dx() * emojiScalar.Dx(),
		Y: imageScalar.Dy() * emojiScalar.Dy(),
	}}

	currentPosition := image.Rectangle{Min: image.Point{0, 0}, Max: emojiScalar.Max}
	emojified := GetTransparent(color.RGBA{}, canvasSize)
	source := Resize(img, imageScalar)

	rand.Seed(time.Now().Unix())
	randomConstantFits := make(map[color.Color]*Emoji)

	LoopPixel(source, func(col []uint8) bool {
		pixColor := BasicToColor(col)

		var bestRandFit *Emoji
		var ok bool

		// this ensures a consistency in colour but still creates a different image each time
		if bestRandFit, ok = randomConstantFits[pixColor]; !ok {

			potentialFits := brand.emojis.colorIndex[brand.emojis.colors.Index(pixColor)]
			bestRandFit = potentialFits[rand.Intn(len(potentialFits))]

			randomConstantFits[pixColor] = bestRandFit
		}

		// translators
		shiftRight := image.Point{emojiScalar.Dx(), 0}
		shiftDown := image.Point{0, emojiScalar.Dy()}

		draw.Draw(emojified, currentPosition, bestRandFit.img, bestRandFit.img.Bounds().Min, draw.Over)

		if currentPosition.Max.X >= canvasSize.Dx() {
			currentPosition = currentPosition.Add(shiftDown)

			currentPosition.Min.X = 0
			currentPosition.Max.X = emojiScalar.Dx()

		} else {
			currentPosition = currentPosition.Add(shiftRight)
		}
		return false
	})

	return emojified, nil
}
