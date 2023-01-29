package main

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif" // for the purpose of this program we only care about the first frame which is what we get
	_ "image/png"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var b64Reg *regexp.Regexp
var experimentalReg *regexp.Regexp

func init() {

	b64Reg, _ = regexp.Compile(".*base64,(.*)")
	experimentalReg, _ = regexp.Compile(`\[(.*)\].*`)
}

type ScrapedResult struct {
	total      int
	errors     []error
	emojiStore EmojiKeg
}

func (scraped *ScrapedResult) Store(s *goquery.Selection, name string, imageOrder int, brandIndex int) error {

	src, state := s.Attr("src")
	if !state {
		return fmt.Errorf("img doesn't contain src")
	}

	b64 := b64Reg.FindStringSubmatch(src)
	if len(b64) == 0 {
		return fmt.Errorf("failed to extract b64")
	}

	dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64[1]))
	img, _, err := image.Decode(dec)
	if err != nil {
		fmt.Println(b64[1])
		return err
	}

	scraped.emojiStore[brandIndex].mu.Lock()
	scraped.emojiStore[brandIndex].emojis.AddAtIndex(name, img, imageOrder)
	scraped.emojiStore[brandIndex].mu.Unlock()

	return nil
}

func (scraped ScrapedResult) getIndex(name string) int {
	for i, el := range scraped.emojiStore {
		if el.name == name {
			return i
		}
	}
	return -1
}

func (scrapedResult *ScrapedResult) AddFromDOM(dom *goquery.Document) (err error) {

	fmt.Print("Parsing")
	table := dom.Find("table tr")

	var brandNames []string

	if brandList := strings.Split(table.Eq(2).Text(), "\n"); len(brandList) > 1 {
		brandNames = brandList[2 : len(brandList)-2] // strip unnecessary information in a future-proof way
	} else {
		return fmt.Errorf("no brandnames found")
	}

	fmt.Print("Extracting")

	relativeTranslation := make(map[int]int) // maps brandNames to master EmojiKeg
	for i, name := range brandNames {

		if scrapedResult.getIndex(name) > -1 {
			relativeTranslation[i] = i
			continue
		}

		scrapedResult.emojiStore = append(scrapedResult.emojiStore, InitBrand(name))
		relativeTranslation[i] = len(scrapedResult.emojiStore) - 1
	}

	old := make([]int, len(scrapedResult.emojiStore)) // ugh, icb explaining this - think about it (translates index as it can be called multiple times)
	for i, el := range scrapedResult.emojiStore {
		old[i] = len(el.emojis.index)
	}

	scraperTotem := struct {
		sync.WaitGroup
		count         int64
		scraperErrors chan error // we have created our own mutex lock - since scraperErrors is a channel of length 1, only one scraper goroutine will right to it once and in extension only one decrements count
	}{count: 0, scraperErrors: make(chan error, 1)}

	primaryScraper := func(emojiIndex int, s *goquery.Selection) {
		var scraperError error

		defer func() {
			scraperTotem.scraperErrors <- scraperError
		}()

		emojis := s.Find(".andr")
		name := s.Find(".name").Text()

		// need to handle cases because their formatting isn't scraper-friendly
		if emojis.Length() == len(scrapedResult.emojiStore) {

			row := s.Find(".andr")
			if row.Length() != len(relativeTranslation) {
				scraperError = fmt.Errorf("malformed row, unexpected number of emojis")
				return
			}

			row.EachWithBreak(func(i int, s *goquery.Selection) bool {

				img := s.Find("img")
				if img.Length() != 1 { // be very strict about this
					return true
				}

				if scraperError = scrapedResult.Store(img, name, emojiIndex+old[relativeTranslation[i]], relativeTranslation[i]); scraperError != nil {
					return false
				}

				scrapedResult.total++
				return true
			}) // normal emoji row

		} else if emojis.Length() == 1 { // this is for new, experimental emojis

			s.Find("img").EachWithBreak(func(i int, s *goquery.Selection) bool {
				scraperError = fmt.Errorf("entry in experimental row didn't have a title")

				title, state := s.Attr("title")
				if !state {
					return false
				}

				titleMatch := experimentalReg.FindStringSubmatch(title)
				if len(titleMatch) == 0 {
					return false
				}
				if titleMatch[1] == "Sample" {
					scraperError = nil // weird loop machanism, not my fault - blame jquery standards
					return true        // we don't care about this
				}

				index := scrapedResult.getIndex(titleMatch[1])
				if index <= -1 {
					return false
				}

				if scraperError = scrapedResult.Store(s, name, emojiIndex+old[relativeTranslation[index]], relativeTranslation[index]); scraperError != nil {
					return false
				}

				scrapedResult.total++
				return true
			})
		}
	}

	table.Each(func(i int, s *goquery.Selection) {
		scraperTotem.count++
		go primaryScraper(i, s)
	})

	var errorMarshal sync.WaitGroup

	errorMarshal.Add(1)
	go func() {
		for scraperTotem.count > 0 {
			if len(scraperTotem.scraperErrors) == 0 {
				continue
			}

			scraperTotem.count-- // this should only be run outside the scraper goroutines
			if erro := <-scraperTotem.scraperErrors; erro != nil {
				scrapedResult.errors = append(scrapedResult.errors, erro) // this does block but we will only be here if at least one goroutine was running
			}
		}
		errorMarshal.Done()
	}()
	errorMarshal.Wait()

	for _, erro := range scrapedResult.errors {
		fmt.Printf("[WARNING] scraper error: %s\t", erro)
	}

	return err // notice that this doesn't include errors from the scraping routine - that's up to the user to decide to look at
}

func Scrape() (result ScrapedResult, err error) {

	website := "https://unicode.org/emoji/charts"
	urls := []string{"full-emoji-list.html", "full-emoji-modifiers.html"}

	for _, page := range urls {

		var resp *http.Response
		// var resp io.Reader
		var doc *goquery.Document

		url := fmt.Sprintf("%s/%s", website, page)
		fmt.Printf("Getting %s\n", url)

		resp, err = http.Get(url)
		// resp, err = os.Open("tests/" + page)
		if err != nil {
			break
		}
		defer resp.Body.Close()
		// defer resp.Close()

		doc, err = goquery.NewDocumentFromReader(resp.Body) // this is so cool! it reads it as it downloads
		if err != nil {
			break
		}

		err = result.AddFromDOM(doc)
	}

	result.emojiStore.StripEmptyEmojis()
	result.emojiStore.PreetifyBrandNames()

	return result, err // maybe add an option to be a bit more lax?
}
