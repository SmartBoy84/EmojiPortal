package main

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"
)

// at time of making, I was getting a total of 24755 emojis - aim for this during future dev

func IsDir(Path string) (bool, error) {
	file, err := os.Open(Path)
	if err != nil {
		return false, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return false, err
	}

	return fileInfo.IsDir(), nil
}

const seperator = "%"

func TestingGround() {

	brand, err := ReadFolder("emojis/Apple", "", Settings{backgroundColor: color.RGBA{A: 255}, imageScale: 0.1})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Cart len %d\n", len(brand.emojis.list))
	err = brand.Emojify("tests/calc.png", "output.png", 1, 100)
	if err != nil {
		panic(err)
	}
	os.Exit(-1)
}

type DstSettings struct {
	mode, pathName          string
	escale, iscale          float64
	quality                 int
	inputImage, outputImage string
}

type SrcSettings struct {
	mode                string
	modifiers           bool
	dirNames, fileNames []string
}

func LoopPathList(paths []string) (filePaths, folderPaths []string) {

	for _, el := range paths {
		nature, err := IsDir(el)
		if err != nil {
			// fmt.Printf("[warning] %s isn't a valid path: %s\n", el, err)
			continue
		}
		if nature {
			folderPaths = append(folderPaths, el)
		} else {
			filePaths = append(filePaths, el)
		}
	}

	return filePaths, folderPaths
}

func extractDst(cmds []string) *DstSettings {

	settings := &DstSettings{escale: 1}
	var err error

	if len(cmds) == 0 {
		return nil
	}

	if cmds[0] == "cart" || cmds[0] == "list" || cmds[0] == "emojify" {
		settings.mode = cmds[0]
		cmds = cmds[1:]
	} else {
		fmt.Println("Didn't specify a mode - cart/list/emojify")
		return nil
	}

	if settings.mode == "emojify" {
		var i int

		for i = range cmds {
			option := strings.Split(cmds[i], ":")

			if len(option) <= 1 {
				continue
			}
			name := option[0]
			value := option[1]

			if name != "escale" && name != "iscale" && name != "quality" { // bear with me
				break
			}
			if name == "escale" || name == "iscale" {

				var scl float64
				if scl, err = strconv.ParseFloat(value, 64); err == nil {
					switch name {
					case "escale":
						settings.escale = scl
					case "iscale":
						settings.iscale = scl
					}
				}
			} else {
				var scl int
				if scl, err = strconv.Atoi(value); err == nil {
					settings.quality = scl
				}
			}

			if err != nil {
				fmt.Printf("[warning] %s specified but error resolving: %s", name, err)
			}
		}

		cmds = cmds[i:]
	}

	filePaths, folderPaths := LoopPathList(cmds)

	if settings.mode == "emojify" {
		if len(filePaths) == 0 || len(filePaths) > 2 {

			if len(folderPaths) > 0 {
				fmt.Print("[warning] tried to input folder path(s) as input/output image path")
			}
			fmt.Print("for emojify mode, specify atleast an input image and at max a second path for output image")
			return nil
		}

		settings.inputImage = filePaths[0]
		if len(filePaths) == 2 {
			settings.outputImage = filePaths[1]
		}
	} else {
		if len(filePaths) == len(cmds) && len(cmds) > 0 {
			fmt.Println("for list/cart, only specify a folder to export them to (maybe you've specified a file path?)")
			return nil
		}
		if len(folderPaths) == 0 {
			switch settings.mode {
			case "cart":
				settings.pathName = "cartridges"
			case "list":
				settings.pathName = "emojis"
			}
		} else {
			settings.pathName = cmds[0]
		}
	}

	return settings
}

func extractSrc(cmds []string) *SrcSettings {

	settings := &SrcSettings{}

	if len(cmds) == 0 {
		settings.mode = "html"
		return settings
	}

	if strings.HasPrefix(cmds[0], "html") {

		if len(cmds) > 1 {
			fmt.Println("html is a standalone argument")
			return nil
		}
		settings.mode = "html"

		if split := strings.Split(cmds[0], ":"); len(split) > 1 {

			if split[1] == "1" {
				settings.modifiers = true
			} else {
				fmt.Println("[warning] html modifier extension specified but malformed (should be 1)")
				return nil
			}
		}

		return settings
	}

	filePaths, folderPaths := LoopPathList(cmds)
	if len(folderPaths) == 0 && len(filePaths) == 0 {
		fmt.Println("no valid files/folders specified")
		return nil
	}
	settings.fileNames = filePaths
	settings.dirNames = folderPaths

	return settings
}

func main() {

	// TestingGround()
	var err error
	var src, dst []string

	for i, el := range os.Args {
		if el == seperator {
			src = os.Args[1:i]
			dst = os.Args[i+1:]
			break
		}
	}

	dstSettings := extractDst(dst)
	srcSettings := extractSrc(src)

	if srcSettings == nil || dstSettings == nil {
		fmt.Printf("\n")
		fmt.Println("For scraping: {folderNames... cartridgeFiles... html{:1 - include modifers} internal} " + seperator + " {[cart/list] {scale:int} {folderName}}\nFor emojifying: {...} % {[convert] {escale:int (emoji scale)} {iscale:int (image scale)} {quality:int} [Source image] {target image}}\n\nensure cartridge files have dimensions at the end of their name as (-XxY)\n*curly braces indicate optional inputs")
		fmt.Printf("\n")

		os.Exit(-1)
	}

	var emojis EmojiKeg

	if srcSettings.mode == "html" {
		results, err := Scrape(srcSettings.modifiers, Settings{imageScale: dstSettings.escale})

		if err != nil {
			panic(err)
		}

		if results.total == 0 {
			panic(fmt.Errorf("no emojis?"))
		}

		emojis = results.emojiStore

	} else {
		var wg sync.WaitGroup

		for _, folderPath := range srcSettings.dirNames {
			wg.Add(1)

			go func(folderPath string) {

				brand, err := ReadFolder(folderPath, "", Settings{imageScale: dstSettings.escale}) // allow custom names for each path
				if err != nil {
					fmt.Println(err)
					return
				}
				emojis = append(emojis, brand)

				wg.Done()
			}(folderPath)
		}

		for _, cartridgePath := range srcSettings.fileNames {
			wg.Add(1)

			go func(cartridgePath string) {

				brand, err := ReadCartridge(cartridgePath, "", 0, 0, Settings{imageScale: dstSettings.escale})
				if err != nil {
					fmt.Println(err)
					return
				}
				emojis = append(emojis, brand)

				wg.Done()
			}(cartridgePath)
		}
		wg.Wait()

		if len(emojis) == 0 {
			panic(fmt.Errorf("no emojis found in folders/cartridges"))
		}
	}

	if dstSettings.mode == "emojify" {

		menu := NewMenu("Pick a brand:")
		index := make(map[string]*Brand)

		for i, brand := range emojis {
			in := fmt.Sprint(i)
			menu.AddItem(fmt.Sprintf("%s - %d emojis", brand.name, len(brand.emojis.list)), in)
			index[in] = brand
		}
		choice := menu.Display()

		err = index[choice].Emojify(dstSettings.inputImage, dstSettings.outputImage, dstSettings.iscale, dstSettings.quality)
		fmt.Println("Emojification complete!")

	} else {
		fmt.Printf("\n%s\n", emojis)

		switch dstSettings.mode {
		case "cart":
			err = emojis.Export(dstSettings.pathName)
		case "list":
			err = emojis.Chunky(dstSettings.pathName)
		}

		fmt.Printf("\n\nSuccessfully scraped and stored emojis!\n")
	}

	if err != nil {
		panic(err)
	}
}
