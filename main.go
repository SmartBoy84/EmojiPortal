package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"
)

const seperator = "%"

//go:embed resources/Apple-72x72.png
var internalBrandBytes []byte

const internalBrandName = "Apple"
const internalBrandX = 72
const internalBrandY = 72

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

type DstSettings struct {
	mode, pathName          string
	escale, iscale          float64
	quality                 float64
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

	settings := &DstSettings{escale: 1, iscale: 1, quality: 1}
	var err error

	if len(cmds) == 0 {
		cmds = append(cmds, "cart") // default value
	}

	if cmds[0] == "cart" || cmds[0] == "list" || cmds[0] == "emojify" {
		settings.mode = cmds[0]
		cmds = cmds[1:]
	} else {
		fmt.Println("Didn't specify a mode - cart/list/emojify")
		return nil
	}

	if settings.mode == "emojify" {
		var x int

		for i := range cmds {
			option := strings.Split(cmds[i], ":")

			if len(option) <= 1 {
				continue
			}

			x++
			name := option[0]
			value := option[1]

			if name != "escale" && name != "iscale" && name != "quality" { // bear with me
				break
			}

			var scl float64
			if scl, err = strconv.ParseFloat(value, 64); err == nil {
				switch name {
				case "escale":
					settings.escale = scl
				case "iscale":
					settings.iscale = scl
				case "quality":
					settings.quality = scl
				}
			} else {
				fmt.Printf("[warning] %s specified but error resolving: %s", name, err)
			}
		}

		cmds = cmds[x:]
	}

	filePaths, folderPaths := LoopPathList(cmds)

	if settings.mode == "emojify" {
		if len(cmds) == 0 || len(cmds) > 2 || len(filePaths) != 1 || len(folderPaths) > 0 {

			if len(folderPaths) > 0 {
				fmt.Printf("[error] tried to input folder path(s) as input/output image path: %s\n", folderPaths)
			}
			if len(filePaths) > 1 {
				fmt.Printf("[error] refuse to overwrite existing file(s): %s\n", filePaths)
			}
			fmt.Print(filePaths, cmds, folderPaths)
			fmt.Println("for emojify mode, specify atleast an input image and at max a second path for output image")
			return nil
		}

		settings.inputImage = filePaths[0]
		if len(cmds) == 2 {
			settings.outputImage = cmds[1]
		}
	} else {
		if len(filePaths) > 0 {
			fmt.Println("for list/cart, only specify a folder to export them to (maybe you've specified a file path?)")
			return nil
		}

		if len(cmds) == 0 {
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

	settings := &SrcSettings{modifiers: true} // default settings
	if len(cmds) == 0 || (len(cmds) == 1 && cmds[0] == "internal") {
		settings.mode = "internal" // don't set it in struct init as then it won't fail when incorrect stuff is specified
		return settings
	}

	if strings.HasPrefix(cmds[0], "html") {

		if len(cmds) > 1 {
			fmt.Println("html is a standalone argument")
			return nil
		}
		settings.mode = "html"

		if split := strings.Split(cmds[0], ":"); len(split) > 1 {

			if split[1] == "0" {
				settings.modifiers = false
			} else {
				fmt.Println("[error] html modifier deactivation specified but malformed (should be 0)")
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

	var err error
	var src, dst []string

	var sepI int

	for i, el := range os.Args {
		if el == seperator {
			src = os.Args[1:i]
			dst = os.Args[i+1:]
			sepI = i // my first hacky solution, is this the end?
			break
		}
	}

	if len(dst) == 0 && sepI == 0 { // check for sepI ensures '%' isn't present
		src = os.Args[1:] // not a mistake, dst comes after the %
	}

	dstSettings := extractDst(dst)
	srcSettings := extractSrc(src)

	if len(os.Args) <= 1 || srcSettings == nil || dstSettings == nil {
		fmt.Println("For scraping: \n{folderNames... cartridgeFiles... html{:0 - exclude modifers} internal} " + seperator + " {[cart/list] {scale:int} {folderName}}\n\nFor emojifying: \n{...} % {emojify {escale:int (emoji scale)} {iscale:int (image scale)} {quality:int} [Source image] {target image}}\n\nensure cartridge files have dimensions at the end of their name as (-XxY)\n*curly braces indicate optional inputs")
		fmt.Printf("\n")

		os.Exit(-1)
	}

	imageSettings := Settings{imageScale: dstSettings.escale}
	if dstSettings.mode == "emojify" {
		imageSettings.backgroundColor = color.RGBA{A: 255}
	}

	var emojis EmojiKeg

	if srcSettings.mode == "internal" {
		internalBrand, err := ReadCartridgeFromBytes(internalBrandBytes, internalBrandName, internalBrandX, internalBrandY, imageSettings)
		if err != nil {
			panic(err)
		}
		emojis = append(emojis, internalBrand)

	} else if srcSettings.mode == "html" {
		results, err := Scrape(srcSettings.modifiers, imageSettings)

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

				brand, err := ReadFolder(folderPath, "", imageSettings) // allow custom names for each path
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

				brand, err := ReadCartridgeFromFile(cartridgePath, "", 0, 0, imageSettings)
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

		var brand *Brand

		if len(emojis) > 1 {

			brandIndex := []*Brand{}
			input := "\n\nSelect brand:\n"

			for i, brand := range emojis {
				name := brand.String()
				brandIndex = append(brandIndex, brand)
				input += fmt.Sprintf("%v. %v\n", i+1, name)
			}

			fmt.Println(input)
			var i int
			for {
				fmt.Printf("Input a number [%d,% d] - ", 1, len(brandIndex))
				fmt.Scan(&i)
				if i > 0 && i <= len(brandIndex) {
					i--
					break
				}
			}

			brand = brandIndex[i]

		} else {
			brand = emojis[0]
		}

		err = brand.Emojify(dstSettings.inputImage, dstSettings.outputImage, dstSettings.iscale, dstSettings.quality)

		if err == nil {
			fmt.Printf("\nEmojification complete!\n")
		}
	} else {
		fmt.Printf("\n%s", emojis)

		switch dstSettings.mode {
		case "cart":
			err = emojis.Export(dstSettings.pathName)
		case "list":
			err = emojis.Chunky(dstSettings.pathName)
		}

		if err == nil {
			fmt.Printf("\nSuccessfully scraped and stored emojis!\n")
		}
	}

	if err != nil {
		panic(err)
	}
}
