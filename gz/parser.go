package gz

import (
	"farma/parser"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	SHORT_SELECTOR string = "[itemtype=\"http://schema.org/Product\"]"
)

var URL string

type productShort struct {
	Thumbnail string `json:"thumbnail"`
	Href      string `json:"href"`
	Title     string `json:"title"`
}

type catalog struct {
	Href         string            `json:"href"`
	Instructions map[string]string `json:"instructions"`
	Shorts       []productShort    `json:"shorts"`
	Analogs      []productShort    `json:"analogs"`
}

type description struct {
	Number     int               `json:"number"`
	Attributes map[string]string `json:"attributes"`
}

type image struct {
	Thumbnail string `json:"thumbnail"`
	SRC       string `json:"src"`
}

type medicament struct {
	Href         string            `json:"href"`
	Title        string            `json:"title"`
	Groups       []string          `json:"groups"`
	Description  *description      `json:"desciption"`
	Features     map[string]string `json:"features"`
	Instructions map[string]string `json:"instructions"`
	Images       []*image          `json:"images"`
	Price        float32           `json:"price"`
}

func letterHrefs(doc *goquery.Document) []string {
	hrefs := []string{}

	doc.Find(".c-alphabet-widget__sign").Each(func(i int, s *goquery.Selection) {
		_, disabled := s.Attr("data-disabled")
		if disabled {
			return
		}

		href, exists := s.Attr("href")
		if exists {
			hrefs = append(hrefs, href)
		}
	})

	return hrefs
}

func catalogHrefs(doc *goquery.Document) []string {
	hrefs := []string{}

	linksList := doc.Find(".c-links-list")
	linksList.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			hrefs = append(hrefs, href)
		}
	})

	return hrefs
}

func newCatalog(href string, doc *goquery.Document) *catalog {
	cat := &catalog{
		Href:         href,
		Instructions: newInstructions(doc, "catalog"),
	}
	cat.Shorts, cat.Analogs = shorts(doc)

	return cat
}

func newInstructions(doc *goquery.Document, pageType string) map[string]string {
	instr := map[string]string{}

	headers := []string{}
	values := []string{}

	var block *goquery.Selection
	switch pageType {
	case "catalog":
		block = doc.Find(".js-aggr-product__anchor[name=\"instructions\"]").Parent().Next()
	case "medicament":
		block = doc.Find("[itemprop=\"description\"]")
		block.Children().First().Remove()
	default:
		log.Fatal("unknown type")
	}

	block.Children().Each(func(i int, s *goquery.Selection) {
		if i%2 == 0 {
			headers = append(headers, s.Find("h3").Text())
		} else {
			values = append(values, strings.TrimSpace(s.Text()))
		}
	})

	for i := range headers {
		instr[headers[i]] = values[i]
	}

	return instr
}

func shorts(doc *goquery.Document) ([]productShort, []productShort) {
	shorts := []productShort{}
	analogs := []productShort{}

	doc.Find(".js-tab-targets").Next().Find(SHORT_SELECTOR).Each(extractShortsHelper(&shorts))
	doc.Find(".js-aggr-product__anchor[name=\"analogs\"]").Parent().Next().Find(SHORT_SELECTOR).Each(extractShortsHelper(&analogs))

	return shorts, analogs
}

func extractShortsHelper(array *[]productShort) func(i int, s *goquery.Selection) {
	return func(i int, s *goquery.Selection) {
		short := &productShort{}

		a := s.Find(".c-prod-item__thumb a")

		thumbnail, exists := a.Find("img").Attr("data-src")
		if exists {
			short.Thumbnail = thumbnail
		}

		href, exists := a.Attr("href")
		if exists {
			short.Href = href
		}

		short.Title = s.Find(".c-prod-item__title").Text()

		*array = append(*array, *short)
	}
}

func newMedicament(href string, doc *goquery.Document) *medicament {
	med := &medicament{
		Href:         href,
		Title:        doc.Find("h1.b-page-title[itemprop=\"name\"]").Text(),
		Groups:       newGroups(doc),
		Description:  newDescription(doc),
		Features:     newFeatures(doc),
		Instructions: newInstructions(doc, "medicament"),
		Images:       newImages(doc),
	}

	priceS := doc.Find("span.js-price-value")
	priceV, err := strconv.ParseFloat(priceS.Text(), 32)
	if err == nil {
		med.Price = float32(priceV)
	}

	return med
}

func newGroups(doc *goquery.Document) []string {
	results := []string{}

	doc.Find("[itemtype=\"http://schema.org/ListItem\"] [itemprop=\"name\"]").Each(func(i int, s *goquery.Selection) {
		results = append(results, strings.TrimSpace(s.Text()))
	})

	if len(results) > 1 {
		return results[1 : len(results)-1]
	}

	return []string{}
}

func newDescription(doc *goquery.Document) *description {
	desc := &description{Attributes: map[string]string{}}

	i, err := strconv.Atoi(doc.Find(".c-product__code .c-product__description").Text())
	if err == nil {
		desc.Number = i
	}

	attrsBlock := doc.Find("div.b-prod-specification div.c-product__specs").First()
	attrsBlock.Find(".c-product__specs-item").Each(func(i int, s *goquery.Selection) {
		attrName := s.Find(".c-product__label").Text()
		desc.Attributes[attrName] = s.Find(".c-product__description").Text()
	})

	return desc
}

func newFeatures(doc *goquery.Document) map[string]string {
	var key string
	features := map[string]string{}

	doc.Find(".c-product-tabs__target-tab table tr td").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())

		if i%2 == 0 {
			key = text
		} else {
			features[key] = text
		}
	})

	return features
}

func newImages(doc *goquery.Document) []*image {
	imgs := []*image{}
	doc.Find(".item.js-product-preview__item").Each(func(i int, s *goquery.Selection) {
		img := &image{}

		val, exists := s.Attr("data-zoom-src")
		if exists {
			img.SRC = val
		}

		val, exists = s.Find("img").Attr("data-src")
		if exists {
			img.Thumbnail = val
		}

		imgs = append(imgs, img)
	})

	return imgs
}

func Jobber(f *parser.FarmaParser) {
	URL = os.Getenv("GZ_URL")

	gottenMedicaments := map[string]bool{}

	for _, letterHref := range letterHrefs(doc(f, "/")) {
		for _, categoryHref := range catalogHrefs(doc(f, letterHref)) {
			catalog := newCatalog(categoryHref, doc(f, categoryHref))
			for _, medicamentShort := range append(catalog.Shorts, catalog.Analogs...) {
				if isIn(medicamentShort.Href, gottenMedicaments) {
					continue
				}

				medicamentDoc := doc(f, medicamentShort.Href)
				medicament := newMedicament(medicamentShort.Href, medicamentDoc)
				f.RawMedicaments <- medicament

				gottenMedicaments[medicamentShort.Href] = true
			}
		}
	}
}

func doc(f *parser.FarmaParser, href string) *goquery.Document {
	u, err := url.Parse(URL)
	if err != nil {
		log.Fatal(err)
	}
	u.Path = path.Join(u.Path, href)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")

	f.Jobs <- &parser.ResponseJob{
		Type:    "doc",
		Request: req,
	}

	rspDoc := <-f.RspDocs
	if rspDoc.Err != nil {
		log.Fatal(err)
	}

	return rspDoc.Doc
}

func isIn(s string, m map[string]bool) bool {
	_, ok := m[s]
	return ok
}
