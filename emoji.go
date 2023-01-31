package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sync"

	"golang.org/x/image/draw"
)

type Settings struct {
	backgroundColor color.RGBA
	imageScale      float64
}

type EmojiKeg []*Brand

type Brand struct {
	mu     sync.Mutex
	name   string
	emojis EmojiStore
}

type EmojiStore struct {
	list       []*Emoji
	colorIndex [][]*Emoji // I chose to this instead of storing indices corresponding to EmojiStore.list as I reasoned they go hand in hand
	colors     color.Palette
}
type Emoji struct {
	name    string
	img     image.Image
	average color.Color
}

var brandTranslations map[string]string

func init() {
	brandTranslations = map[string]string{
		"BrowserAppl": "Apple", // yeah this one is a bug in goquery - whatever
		"Goog":        "Google",
		"FB":          "Facebook",
		"Wind":        "Windows",
		"Twtr":        "Twitter",
		"Sams":        "Samsung",
		"GMail":       "Gmail",
		"SB":          "SB",
		"DCM":         "DCM",
		"KDDI":        "KDDI",
		"Joy":         "Joy",
	}
}

// false - continue, true - break
func LoopPixel(img image.Image, cb func(col []uint8) bool) bool {

	bounds := img.Bounds()
	rgba := GetTransparent(color.RGBA{}, bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Over)

	index := 0
	var col []uint8
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			col = rgba.Pix[index : index+4]
			if cb(col) {
				return true
			}
			index += 4
		}
	}
	return false
}

func (emoji *Emoji) GetAverage() color.Color {

	colorSum := make([]uint64, 4)

	LoopPixel(emoji.img, func(col []uint8) bool {
		if col[3] > 0 {
			for i, el := range col {
				colorSum[i] += uint64(el)
			}
		}
		return false
	})

	imgSize := emoji.img.Bounds().Dx() * emoji.img.Bounds().Dy()
	averageColor := make([]uint8, 4)
	for i := range colorSum {
		averageColor[i] = uint8(math.Round(float64(colorSum[i]) / float64(imgSize)))
	}
	averageColor[3] = 255
	// return color.RGBA{R: averageColor[0], G: averageColor[1], B: averageColor[2], A: 0}
	return BasicToColor(averageColor)
}

func GetTransparent(col color.Color, dimensions image.Rectangle) *image.RGBA {

	canvas := image.NewRGBA(dimensions)
	draw.Draw(canvas, canvas.Bounds(),
		&image.Uniform{C: col}, // color.RGBA{} = color.RGBA{R:0, G:0, B:0, A:0} => A:0 = transparent
		image.Point{}, draw.Src)

	return canvas
}

func CreateEmoji(name string, img image.Image) *Emoji {
	emoji := Emoji{img: img, average: color.RGBA{}, name: name}
	emoji.average = emoji.GetAverage()
	return &emoji
}

// func (store *EmojiStore) Push(name string, img image.Image) {
// 	emoji := CreateEmoji(name, img)
// }

func (store *EmojiStore) Add(name string, img image.Image, i int) {

	emoji := CreateEmoji(name, img)

	if i > -1 {
		if i+1 > len(store.list) {
			store.list = append(store.list, make([]*Emoji, i+1-len(store.list))...)
		}

		store.list[i] = emoji
	} else {
		store.list = append(store.list, emoji)
	}

	for i, col := range store.colors {
		if col == emoji.average {
			store.colorIndex[i] = append(store.colorIndex[i], emoji)
			return
		}
	}

	store.colors = append(store.colors, emoji.average)
	store.colorIndex = append(store.colorIndex, []*Emoji{emoji})
}

func InitBrand(name string) *Brand {
	return &Brand{name: name}
}

func CreateScalar(img image.Image, scale float64) (image.Rectangle, error) {
	if scale <= 0 || scale > 1 {
		return image.Rectangle{}, fmt.Errorf("resolution must be (0, 1]: %v", scale)
	}

	return image.Rect(0, 0,
		int(math.Ceil(float64(img.Bounds().Max.X)*scale)),
		int(math.Ceil(float64(img.Bounds().Max.Y)*scale)),
	), nil
}

func (brand *Brand) GetScalar(scale float64) (image.Rectangle, error) {

	var some *Emoji
	for _, some = range brand.emojis.list {
		scalar, err := CreateScalar(some.img, scale)

		if err != nil {
			return image.Rectangle{}, err
		} else {
			return scalar, nil
		}
	}

	return image.Rectangle{}, fmt.Errorf("emoji list is empty")
}

func (emojis EmojiKeg) PreetifyBrandNames() {
	for i := range emojis {
		if actualName, exists := brandTranslations[emojis[i].name]; exists {
			emojis[i].name = actualName
		}
	}
}

func (emojis EmojiKeg) String() string {
	list := ""
	total := 0
	for _, brand := range emojis {
		list += brand.String()
		total += len(brand.emojis.list)
	}
	if len(emojis) > 1 {
		list += fmt.Sprintf("total - %d", total)
	}
	return list
}

func (brand *Brand) String() string {
	return fmt.Sprintf("%v - %v emojis", brand.name, len(brand.emojis.list))
}

func ColorToBasic(col color.Color) []uint8 {
	r, g, b, a := col.RGBA()
	return []uint8{uint8(r), uint8(g), uint8(b), uint8(a)}
}

func BasicToColor(col []uint8) color.Color {
	if len(col) != 4 {
		fmt.Print("Warning, malformed color")
		return nil
	}
	return color.RGBA{R: col[0], G: col[1], B: col[2], A: col[3]}
}

func (brand *Brand) CleanUp() { // mainly for: 1. reading cartridges (there will definitely be black strips at the end), 2. getting emojis from the internet (trust me, this is the best solution)

	/*
	   On the site, the emojis are ordered in a table
	   In each row, the corresponding emoji type for each brand can potentially be missing
	   My program doesn't notice that and the index it uses is the overall index
	   This results in the emoji index having massing gaps where there were missing emojis

	 trust me, the solution below the best one for this way of scraping the emojis
	*/

	newList := []*Emoji{}
	for _, el := range brand.emojis.list {
		if el != nil && el.img != nil {
			newList = append(newList, el)
		}
	}
	// fmt.Printf("after %d", len(brand.emojis.index))

	// now, let's clean up any emojis of uniform colour
	deleted := 0

	for i, emoji := range newList {

		col := ColorToBasic(emoji.img.At(0, 0))

		if LoopPixel(emoji.img, func(target []uint8) bool {
			for i, c := range target {
				if c != col[i] {
					return true
				}
			}
			return false
		}) {
			continue
		}

		// fmt.Printf("Snipped %s for %s\n", brand.emojis.list[i].name, brand.name)
		newList = append(newList[:i-deleted], newList[i+1-deleted:]...)
		deleted++
	}

	brand.emojis.list = newList
}

func (keg EmojiKeg) StripEmptyEmojis() {
	for _, brand := range keg {
		brand.CleanUp()
	}
}

func ApplySettings(img image.Image, imageSettings Settings) (image.Image, error) {
	scalar, err := CreateScalar(img, imageSettings.imageScale)
	if err != nil {
		return nil, err
	}

	imageData := Resize(img, scalar)

	emoji := GetTransparent(imageSettings.backgroundColor, image.Rectangle{
		image.Point{0, 0},
		imageData.Bounds().Max,
	})

	draw.Draw(emoji, emoji.Bounds(), imageData, image.Point{0, 0}, draw.Over)
	return imageData, nil
}

func Resize(emoji image.Image, scalar image.Rectangle) image.Image {

	if scalar.Dx() == emoji.Bounds().Dx() && scalar.Dy() == emoji.Bounds().Dy() {
		return emoji // image already 100%
	}

	dst := image.NewRGBA(scalar)
	draw.NearestNeighbor.Scale(dst, scalar, emoji, emoji.Bounds(), draw.Src, nil)

	return dst
}
