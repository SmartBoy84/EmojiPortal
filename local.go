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

func ReadFolder(folderPath string, brandName string) (*Brand, error) {

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

		file, err := os.Open(fmt.Sprintf("%s/%s", folderPath, f.Name()))
		if err != nil {
			fmt.Print(err)
			continue
		}
		defer file.Close()

		imageData, _, err := image.Decode(file)
		if err != nil {
			fmt.Print(err)
			continue
		}
		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))

		id := strings.Split(name, "__")
		if len(id) == 2 {
			var i int

			i, err = strconv.Atoi(id[0])
			if err == nil {
				brand.emojis.AddAtIndex(id[1], imageData, i)
				continue
			}
		}

		fmt.Printf("image successfully read but no id present in name => {[index]__[name]}")
		brand.emojis.Add(name, imageData)
	}

	brand.CleanUp()
	return brand, nil
}

func ReadCartridge(fileName string, brandName string, X int, Y int) (*Brand, error) {

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

	brand := InitBrand(brandName)
	fmt.Printf("Making emojikeg from %s\n", fileName)

	cartridgeFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer cartridgeFile.Close()

	imageData, _, err := image.Decode(cartridgeFile)
	if err != nil {
		return brand, err
	}

	cartridgeSize := imageData.Bounds().Max
	emojiSize := image.Rect(0, 0, X, Y)
	currentPosition := image.Rectangle{Min: image.Point{0, 0}, Max: emojiSize.Max}

	// translators
	shiftRight := image.Point{emojiSize.Dx(), 0}
	shiftDown := image.Point{0, emojiSize.Dy()}

	for i := 0; ; i++ {

		emoji := GetTransparent(image.Rectangle{
			image.Point{0, 0},
			emojiSize.Max,
		})

		draw.Draw(emoji, emojiSize, imageData, currentPosition.Min, draw.Src)
		brand.emojis.AddAtIndex(fmt.Sprintf("%d", i), emoji, i)

		if currentPosition.Max.X >= cartridgeSize.X {
			currentPosition = currentPosition.Add(shiftDown)
			if currentPosition.Min.Y >= imageData.Bounds().Dy() {
				break
			}

			currentPosition.Min.X = 0
			currentPosition.Max.X = emojiSize.Dx()

		} else {
			currentPosition = currentPosition.Add(shiftRight)
		}
	}

	brand.CleanUp()
	return brand, nil
}
