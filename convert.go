package main

import (
	"image"
	"image/color"
	_ "image/gif" // for the purpose of this program we only care about the first frame which is what we get
	_ "image/jpeg"
	_ "image/png"
	"math/rand"
	"time"

	"golang.org/x/image/draw"
)

func (brand *Brand) Emojify(inputName string, outputName string, imageScale float64, emojiScale float64) error {
	imageData, err := OpenImage(inputName)
	if err != nil {
		return err
	}

	img, err := brand.ConvertImage(imageData, imageScale, emojiScale)
	if err != nil {
		return err
	}

	if err := Export(outputName, img); err != nil {
		return err
	}

	return nil
}

func (brand *Brand) ConvertImage(img image.Image, imageScale, emojiScale float64) (image.Image, error) {

	imageScalar, err := CreateScalar(img, imageScale)
	if err != nil {
		return nil, err
	}

	emojiScalar, err := brand.GetScalar(emojiScale) // not possible to get an errr here
	if err != nil {
		return nil, err
	}

	canvasSize := image.Rectangle{Max: image.Point{
		X: imageScalar.Dx() * emojiScalar.Dx(),
		Y: imageScalar.Dy() * emojiScalar.Dy(),
	}}

	currentPosition := image.Rectangle{Min: image.Point{0, 0}, Max: emojiScalar.Max}
	emojified := GetTransparent(canvasSize)
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
