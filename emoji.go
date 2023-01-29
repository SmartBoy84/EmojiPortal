package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path"
	"sync"

	"golang.org/x/image/draw"
)

type EmojiKeg []*Brand

type Brand struct {
	mu     sync.Mutex
	name   string
	emojis EmojiStore
}

type EmojiStore struct {
	index []string
	list  map[string]Emoji
}
type Emoji image.Image

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

func GetTransparent(dimensions image.Rectangle) *image.RGBA {

	canvas := image.NewRGBA(dimensions)
	draw.Draw(canvas, canvas.Bounds(),
		&image.Uniform{C: color.RGBA{}}, // color.RGBA{} = color.RGBA{R:0, G:0, B:0, A:0} => A:0 = transparent
		image.Point{}, draw.Src)

	return canvas
}

func (store *EmojiStore) Add(name string, img image.Image) {
	store.index = append(store.index, name)
	store.list[name] = img
}

func (store *EmojiStore) AddAtIndex(name string, img image.Image, i int) {

	if i+1 > len(store.index) {
		store.index = append(store.index, make([]string, i+1-len(store.index))...)
	}

	store.index[i] = name
	store.list[name] = img
}

func (store *EmojiStore) Delete(name string) {
	for i, el := range store.index {
		if el == name {
			delete(store.list, name)
			store.index[i] = store.index[len(store.index)-1]
			store.index = store.index[:len(store.index)-1]
		}
	}
}

func (store *EmojiStore) DeleteAtIndex(n int) {
	if n < len(store.index) {
		delete(store.list, store.index[n])
	}
}

func InitBrand(name string) *Brand {
	return &Brand{
		name: name,
		emojis: EmojiStore{
			index: []string{},
			list:  map[string]Emoji{}},
	}
}

func (brand *Brand) GetScalar(percentScale int) (image.Rectangle, error) {

	if percentScale <= 0 || percentScale > 100 {
		return image.Rectangle{}, fmt.Errorf("resolution must be (0, 100]")
	}

	var some Emoji
	for _, some = range brand.emojis.list {
		break
	}

	return image.Rect(0, 0,
		some.Bounds().Max.X*percentScale/100,
		some.Bounds().Max.Y*percentScale/100,
	), nil
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
		list += fmt.Sprintf("%s - %d emojis\n", brand.name, len(brand.emojis.list))
		total += len(brand.emojis.list)
	}
	return list + fmt.Sprintf("total - %d", total)
}

func (brand *Brand) CleanUp() { // mainly for: 1. reading cartridges (there will definitely be black strips at the end), 2. getting emojis from the internet (trust me, this is the best solution)

	/*
	   On the site, the emojis are ordered in a table
	   In each row, the corresponding emoji type for each brand can potentially be missing
	   My program doesn't notice that and the index it uses is the overall index
	   This results in the emoji index having massing gaps where there were missing emojis

	 trust me, the solution below the best one for this way of scraping the emojis
	*/

	// fmt.Printf("before %d", len(brand.emojis.index))
	newList := []string{}
	for _, el := range brand.emojis.index {
		if len(el) > 0 {
			newList = append(newList, el)
		}
	}
	brand.emojis.index = newList
	// fmt.Printf("after %d", len(brand.emojis.index))

	// now, let's clean up any emojis of uniform colour
	for name, emoji := range brand.emojis.list {

		r, g, b, a := emoji.At(0, 0).RGBA()

		for x := 0; x < emoji.Bounds().Dx(); x++ {
			for y := 0; y < emoji.Bounds().Dy(); y++ {

				nr, ng, nb, na := emoji.At(x, y).RGBA()
				if nr != r || ng != g || nb != b || na != a {
					goto moveon
				}
			}
		}

		// fmt.Print("Snip!")
		brand.emojis.Delete(name)

	moveon:
		continue
	}
}

func (keg EmojiKeg) StripEmptyEmojis() {
	for _, brand := range keg {
		brand.CleanUp()
	}
}

func Resize(emoji image.Image, scalar image.Rectangle) (image.Image, error) {

	if scalar.Dx() == emoji.Bounds().Dx() && scalar.Dy() == emoji.Bounds().Dy() {
		return emoji, nil // image already 100%
	}

	dst := image.NewRGBA(scalar)
	draw.NearestNeighbor.Scale(dst, scalar, emoji, emoji.Bounds(), draw.Src, nil)

	return dst, nil
}

func (emojis EmojiKeg) Chunky(folderName string, percentScale int) error { // depecrated, only use cartridges

	for _, brand := range emojis {
		if err := brand.ExportEmojis(fmt.Sprintf("%s/%s", folderName, brand.name), percentScale); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) ExportEmojis(folderName string, percentScale int) error {

	fmt.Printf("\nExporting emojis for %s", brand.name)

	var scalar image.Rectangle
	var err error

	if scalar, err = brand.GetScalar(percentScale); err != nil {
		return err
	}

	if err := os.MkdirAll(folderName, 0700); err != nil {
		return err
	}

	for i, name := range brand.emojis.index {
		i++
		img, err := Resize(brand.emojis.list[name], scalar)
		if err != nil {
			return err
		}

		picBuff := bytes.Buffer{}

		if err := png.Encode(&picBuff, img); err != nil {
			return err
		}

		if err := os.WriteFile(fmt.Sprintf("%s/%d__%s.png", folderName, i, name), picBuff.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (emojis EmojiKeg) Export(folderName string, percentScale int) error {

	for _, brand := range emojis {
		if err := brand.CreateCartridge(fmt.Sprintf("%s/%s", folderName, brand.name), percentScale); err != nil {
			return err
		}
	}
	return nil
}

func (brand *Brand) CreateCartridge(fileName string, percentScale int) error {

	fmt.Printf("\nCreating cartridge for %s", brand.name)

	var err error
	var scalar image.Rectangle

	if scalar, err = brand.GetScalar(percentScale); err != nil {
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
	Shift := func(vector image.Point) {
		currentPosition.Min.X += vector.X
		currentPosition.Max.X += vector.X
		currentPosition.Min.Y += vector.Y
		currentPosition.Max.Y += vector.Y
	}

	// translators
	shiftRight := image.Point{scalar.Dx(), 0}
	shiftDown := image.Point{0, scalar.Dy()}

	for _, name := range brand.emojis.index {

		if len(name) == 0 {
			continue
		}

		scaledEmoji, err := Resize(brand.emojis.list[name], scalar)
		if err != nil {
			return err
		}

		draw.Draw(canvas, currentPosition, scaledEmoji, image.Point{0, 0}, draw.Over)

		if currentPosition.Max.X == canvasSize.X {

			Shift(shiftDown)

			currentPosition.Min.X = 0
			currentPosition.Max.X = scalar.Dx()

		} else {
			Shift(shiftRight)
		}
	}

	out, err := os.Create(fmt.Sprintf("%s-%dx%d.png", fileName, scalar.Dx(), scalar.Dy()))
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	png.Encode(out, canvas)
	return nil
}
