package gz

import (
	"farma/mongodb"
	"farma/parser"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	SHORT_SELECTOR string = "[itemtype=\"http://schema.org/Product\"]"
)

var URL string = string([]byte{104, 116, 116, 112, 115, 58, 47, 47, 103, 111, 114, 122, 100, 114, 97, 118, 46, 111, 114, 103, 47})

type productShort struct {
	Thumbnail string `json:"thumbnail"`
	Path      string `json:"path"`
	Title     string `json:"title"`
}

type catalog struct {
	Path         string            `json:"path"`
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
	Path         string            `json:"path"`
	Title        string            `json:"title"`
	Groups       []string          `json:"groups"`
	Description  *description      `json:"desciption"`
	Features     map[string]string `json:"features"`
	Instructions map[string]string `json:"instructions"`
	Images       []*image          `json:"images"`
	Price        float32           `json:"price"`
}

func letterPaths(doc *goquery.Document) []string {
	paths := []string{}

	doc.Find(".c-alphabet-widget__sign").Each(func(i int, s *goquery.Selection) {
		_, disabled := s.Attr("data-disabled")
		if disabled {
			return
		}

		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	return paths
}

func catalogPaths(doc *goquery.Document) []string {
	paths := []string{}

	linksList := doc.Find(".c-links-list")
	linksList.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	return paths
}

func newCatalog(path string, doc *goquery.Document) *catalog {
	cat := &catalog{
		Path:         path,
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

		thumbnail, tExists := a.Find("img").Attr("data-src")
		if tExists {
			short.Thumbnail = thumbnail
		}

		path, lExists := a.Attr("href")
		if lExists {
			short.Path = path
		}

		short.Title = s.Find(".c-prod-item__title").Text()

		*array = append(*array, *short)
	}
}

func newMedicament(path string, doc *goquery.Document) *medicament {
	med := &medicament{
		Path:         path,
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

func extract(p *parser.Parser, limit int, meds chan *medicament, cats chan *catalog, quit chan bool) {
	mapped := map[string]bool{}

	for _, letP := range letterPaths(doc(p, "/")) {
		for _, catP := range catalogPaths(doc(p, letP)) {
			cat := newCatalog(catP, doc(p, catP))
			cats <- cat
			for _, sh := range append(cat.Shorts, cat.Analogs...) {
				if isIn(sh.Path, mapped) {
					continue
				}

				meds <- newMedicament(sh.Path, doc(p, sh.Path))
				mapped[sh.Path] = true

				if len(mapped) == limit {
					quit <- true
					return
				}
			}
		}
	}

	quit <- true
}

func doc(p *parser.Parser, path string) *goquery.Document {
	p.Paths <- path
	return (<-p.Docs).Doc
}

func isIn(s string, m map[string]bool) bool {
	_, ok := m[s]
	return ok
}

func insertAll(medColName, catColName string, meds chan *medicament, cats chan *catalog, quit chan bool) {
	mongoClient := mongodb.NewMongoClient()
	for {
		select {
		case m := <-meds:
			mongoClient.InsertOne(medColName, m)
		case c := <-cats:
			mongoClient.InsertOne(catColName, c)
		case <-quit:
			return
		}
	}
}

func ParseAll(medColName, catColName string) {
	prsr := &parser.Parser{
		Paths: make(chan string, 1),
		Docs:  make(chan *parser.Doc, 1),
		Pause: 2 * time.Second,
		URL:   URL,
	}
	meds := make(chan *medicament, 1)
	cats := make(chan *catalog, 1)
	quit := make(chan bool)

	go prsr.Parse()
	go insertAll(medColName, catColName, meds, cats, quit)

	extract(prsr, -1, meds, cats, quit)
}
