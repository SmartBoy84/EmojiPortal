package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// at time of making, I was getting a total of 24755 emojis - aim for this during future dev

func main() {
	var src, dest []string

	for i, el := range os.Args {
		if el == "%" {
			src = os.Args[1:i]
			dest = os.Args[i+1:]
			break
		}
	}

	if len(src) == 0 || len(dest) < 2 || !(dest[0] == "cart" || dest[0] == "list") {
		fmt.Println("[folderNames... cartridgeFiles... html{:1 - include modifers}] % [cart/list] {scale:int} [folderName]\nensure cartridge files have dimensions at the end of their name as (-XxY)")
		os.Exit(-1)
	}

	scale := 100

	nameOpt := strings.Split(dest[1], "scale:")
	if len(nameOpt) == 2 {
		if scl, err := strconv.Atoi(nameOpt[1]); err == nil {
			scale = scl
			dest = []string{dest[0], dest[2]}
		} else {
			fmt.Printf("[warning] scale specified but error resolving: %s", err)
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

			file, err := os.Open(el)
			if err != nil {
				fmt.Printf("[warning] %s isn't a valid path\n", el)
				continue
			}

			fileInfo, err := file.Stat()
			if err != nil {
				fmt.Print(err)
				continue
			}

			if fileInfo.IsDir() {
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

	fmt.Println(emojis)

	if dest[0] == "cart" {
		if err := emojis.Export(dest[1], scale); err != nil {
			panic(err)
		}
	} else if dest[0] == "list" {
		if err := emojis.Chunky(dest[1], scale); err != nil {
			panic(err)
		}
	} else {
		panic(fmt.Errorf("destination must be prefixed with cart/list"))
	}
	fmt.Printf("\n\nSuccessfully scraped and stored emojis!\n")
}
