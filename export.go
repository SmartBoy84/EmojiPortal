package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path"

	"golang.org/x/image/draw"
)

func Export(fileName string, img image.Image) error {
	out, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return err
	}

	return nil
}

func (emojis EmojiKeg) Chunky(folderName string, scale float64) error { // depecrated, only use cartridges

	for _, brand := range emojis {
		if err := brand.ExportEmojis(fmt.Sprintf("%s/%s", folderName, brand.name), scale); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) ExportEmojis(folderName string, scale float64) error {

	fmt.Printf("\nExporting emojis for %s", brand.name)

	var scalar image.Rectangle
	var err error

	if scalar, err = brand.GetScalar(scale); err != nil {
		return err
	}

	if err := os.MkdirAll(folderName, 0700); err != nil {
		return err
	}

	for i, emoji := range brand.emojis.list {
		img := Resize(emoji.img, scalar)

		if err := Export(fmt.Sprintf("%s/%d__%s.png", folderName, i, emoji.name), img); err != nil {
			return err
		}
	}

	return nil
}

func (emojis EmojiKeg) Export(folderName string, scale float64) error {

	for _, brand := range emojis {
		if err := brand.CreateCartridge(fmt.Sprintf("%s/%s", folderName, brand.name), scale); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) CreateCartridge(fileName string, scale float64) error {

	fmt.Printf("\nCreating cartridge for %s", brand.name)

	var err error
	var scalar image.Rectangle

	if scalar, err = brand.GetScalar(scale); err != nil {
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

	canvas := GetTransparent(image.Rectangle{
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

	if err := Export(fmt.Sprintf("%s-%dx%d.png", fileName, scalar.Dx(), scalar.Dy()), canvas); err != nil {
		return err
	}
	return nil
}
