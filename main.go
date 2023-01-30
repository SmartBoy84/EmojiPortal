package main

import (
	"fmt"
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
		fmt.Print(err)
		return false, err
	}

	return fileInfo.IsDir(), nil
}

const seperator = "%"

func main() {

	var src, dst []string
	dst = []string{"list"}

	for i, el := range os.Args {
		if el == seperator {
			src = os.Args[1:i]
			dst = os.Args[i+1:]
			break
		}
	}

	if len(os.Args) == 1 || !(len(dst) == 0 || dst[0] == "cart" || dst[0] == "list") {
		fmt.Println("{folderNames... cartridgeFiles... html{:1 - include modifers}} " + seperator + " {[cart/list] {scale:int} {folderName}}\n\nensure cartridge files have dimensions at the end of their name as (-XxY)\n*curly braces indicate optional inputs")
		fmt.Printf("\n")
		os.Exit(-1)
	}

	if len(src) == 0 {

		if os.Args[1] == seperator || len(src) == 0 {
			src = []string{"html"}
			fmt.Printf("%s\n%s", src, dst)
		} else {
			src = os.Args[1:] // this is all to allow user to just specify a src
		}
	}

	scale := float64(1)
	if len(dst) > 1 {
		nameOpt := strings.Split(dst[1], "scale:")
		if len(nameOpt) == 2 {
			if scl, err := strconv.ParseFloat(nameOpt[1], 64); err == nil {
				scale = scl
				dst = append(dst[:1], dst[2:]...)
			} else {
				fmt.Printf("[warning] scale specified but error resolving: %s", err)
			}
		}
	}

	var emojis EmojiKeg

	if len(src) == 1 && strings.HasPrefix(src[0], "html") {

		IncludeModifiers := false
		if split := strings.Split(src[0], ":"); len(split) > 1 {
			if scl, err := strconv.Atoi(split[1]); err == nil && scl == 1 {
				IncludeModifiers = true
			} else {
				fmt.Println("[warning] include modifers option ignored as it should only be html:1")
			}
		}
		results, err := Scrape(IncludeModifiers)

		if err != nil {
			panic(err)
		}

		if results.total == 0 {
			panic(fmt.Errorf("no emojis?"))
		}

		emojis = results.emojiStore

	} else {
		var folders, files []string

		for _, el := range src {
			nature, err := IsDir(el)
			if err != nil {
				fmt.Printf("[warning] %s isn't a valid path: %s\n", el, err)
				continue
			}
			if nature {
				folders = append(folders, el)
			} else {
				files = append(files, el)
			}
		}

		if len(folders) == 0 && len(files) == 0 {
			panic(fmt.Errorf("no valid files/folders specified"))
		}

		var wg sync.WaitGroup

		for _, folderPath := range folders {
			wg.Add(1)

			go func(folderPath string) {

				brand, err := ReadFolder(folderPath, "") // allow custom names for each path
				if err != nil {
					fmt.Print(err)
					return
				}
				emojis = append(emojis, brand)

				wg.Done()
			}(folderPath)
		}

		for _, cartridgePath := range files {
			wg.Add(1)

			go func(cartridgePath string) {

				brand, err := ReadCartridge(cartridgePath, "", 0, 0)
				if err != nil {
					fmt.Print(err)
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

	fmt.Printf("\n%s\n", emojis)

	destinationName := ""
	if len(dst) >= 2 {
		nature, err := IsDir(dst[1])
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("[warning] destination path specified but error resolving: %s\n", err)
		} else {
			if nature {
				fmt.Println("[warning] destination path specified but not a folder")
			} else {
				destinationName = dst[1]
			}
		}
	}

	if dst[0] == "cart" {
		if destinationName == "" {
			destinationName = "cartridges"
		}

		if err := emojis.Export(destinationName, scale); err != nil {
			panic(err)
		}
	} else if dst[0] == "list" {
		if destinationName == "" {
			destinationName = "emojis"
		}

		if err := emojis.Chunky(destinationName, scale); err != nil {
			panic(err)
		}
	} else {
		panic(fmt.Errorf("destination must be prefixed with cart/list"))
	}
	fmt.Printf("\n\nSuccessfully scraped and stored emojis!\n")
}
