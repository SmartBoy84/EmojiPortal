package main

import (
	"fmt"
	"image"
	_ "image/gif" // for the purpose of this program we only care about the first frame which is what we get
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

func OpenImage(fileName string) (image.Image, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	imageData, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}

func ReadFolder(folderPath string, brandName string, imageSettings Settings) (*Brand, error) {

	var err error

	if len(brandName) == 0 {
		brandName = filepath.Base(folderPath)
	}

	brand := InitBrand(brandName)
	fmt.Printf("Making emojikeg from images in %s\n", folderPath)

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return brand, err
	}

	for _, f := range files {

		imageData, err := OpenImage(fmt.Sprintf("%s/%s", folderPath, f.Name()))
		if err != nil {
			fmt.Print(err)
			continue
		}

		emoji, err := ApplySettings(imageData, imageSettings)
		if err != nil {
			return nil, err
		}

		
		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))

		id := strings.Split(name, "__")
		if len(id) == 2 {
			var i int

			i, err = strconv.Atoi(id[0])
			if err == nil {
				brand.emojis.Add(id[1], emoji, i)
				continue
			}
		}

		fmt.Printf("image successfully read but no id present in name => {[index]__[name]}")
		brand.emojis.Add(name, emoji, -1)
	}

	brand.CleanUp()
	return brand, nil
}

func ReadCartridge(fileName string, brandName string, X int, Y int, imageSettings Settings) (*Brand, error) {

	test := regexp.MustCompile(`(.*)-(\d*)x(\d*)$`)

	if len(brandName) == 0 {
		brandName = filepath.Base(fileName)
		brandName = strings.TrimSuffix(brandName, filepath.Ext(brandName))
	}

	if X == 0 || Y == 0 {

		match := test.FindStringSubmatch(brandName)

		if len(match) != 4 { // [full, suffix, X, Y]
			return nil, fmt.Errorf("no dimensions specified and failed to infer from name (must be suffixed with -XxY)")
		}

		brandName = match[1]
		match = match[2:]

		var conversion []int
		for _, el := range match {
			i, err := strconv.Atoi(el)
			if err != nil {
				return nil, fmt.Errorf("error while resolving dimension: %s", err)
			}
			conversion = append(conversion, i)
		}

		X = conversion[0]
		Y = conversion[1]
	}

	emojiScalar, err := CreateScalar(image.Rectangle{Max: image.Point{X, Y}}, imageSettings.imageScale)
	if err != nil {
		return nil, err
	}

	brand := InitBrand(brandName)
	fmt.Printf("Making emojikeg from %s\n", fileName)

	imageData, err := OpenImage(fileName)
	if err != nil {
		return nil, err
	}
	imageScalar, _ := CreateScalar(imageData, imageSettings.imageScale)
	imageData = Resize(imageData, imageScalar)

	cartridgeSize := imageData.Bounds().Max
	currentPosition := image.Rectangle{Min: image.Point{0, 0}, Max: emojiScalar.Max}

	// translators
	shiftRight := image.Point{emojiScalar.Dx(), 0}
	shiftDown := image.Point{0, emojiScalar.Dy()}

	for i := 0; ; i++ {

		emoji := GetTransparent(imageSettings.backgroundColor, image.Rectangle{
			image.Point{0, 0},
			emojiScalar.Max,
		})

		draw.Draw(emoji, emojiScalar, imageData, currentPosition.Min, draw.Src)
		brand.emojis.Add(fmt.Sprintf("%d", i), emoji, i)

		if currentPosition.Max.X >= cartridgeSize.X {
			currentPosition = currentPosition.Add(shiftDown)
			if currentPosition.Min.Y >= imageData.Bounds().Dy() {
				break
			}

			currentPosition.Min.X = 0
			currentPosition.Max.X = emojiScalar.Dx()

		} else {
			currentPosition = currentPosition.Add(shiftRight)
		}
	}

	brand.CleanUp()
	return brand, nil
}
