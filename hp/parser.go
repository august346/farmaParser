package hp

import (
	"encoding/json"
	"farma/mongodb"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	PATH_MNN string = "/ingredients/"
)

var URL string = string([]byte{104, 116, 116, 112, 115, 58, 47, 47, 112, 108, 97, 110, 101, 116, 97, 122, 100, 111, 114, 111, 118, 111, 46, 114, 117})

type subAttribute struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type attribute struct {
	Name          string          `json:"name"`
	SubAttributes []*subAttribute `json:"subAttribute"`
}

type medicament struct {
	Path        string            `json:"path"`
	Groups      []string          `json:"groups"`
	Title       string            `json:"title"`
	Price       float32           `json:"price"`
	Images      []string          `json:"images"`
	Features    map[string]string `json:"features"`
	AnalogPaths []string          `json:"analogPaths"`
	Attributes  []*attribute      `json:"attributes"`
}

func letterPaths(doc *goquery.Document) []string {
	paths := []string{}

	doc.Find("li.main-alphabet__nav-item a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	return paths
}

func mnnPaths(doc *goquery.Document) []string {
	paths := []string{}

	doc.Find(".main-alphabet__list a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	return paths
}

func medicamentPaths(doc *goquery.Document, ticker *time.Ticker, withNexts bool) []string {
	paths := []string{}

	doc.Find("div.card-list__element a.product-card__image").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	if !withNexts {
		return paths
	}

	doc.Find("div.pagination.pagination_large a.pagination__item").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, medicamentPaths(newDoc(href, nil, ticker), ticker, false)...)
		}
	})

	return paths
}

func newMedicament(path string, doc *goquery.Document) *medicament {
	med := &medicament{
		Path:        path,
		Title:       strings.TrimSpace(doc.Find("h1.product-detail__title").Text()),
		Groups:      groups(doc),
		Images:      images(doc),
		Features:    features(doc),
		AnalogPaths: analogPaths(doc),
		Attributes:  attributes(doc),
	}

	priceDiv := doc.Find("div.product-detail__price_new")
	priceDivId, exists := priceDiv.Attr("id")
	if exists {
		price, err := strconv.ParseFloat(priceDivId, 32)
		if err == nil {
			med.Price = float32(price)
		}
	}

	return med
}

func groups(doc *goquery.Document) []string {
	results := []string{}

	doc.Find(".nav-bread-crumbs__item a").Each(func(i int, s *goquery.Selection) {
		title, exists := s.Attr("title")
		if exists {
			results = append(results, title)
		}
	})

	if len(results) > 2 {
		return results[2:]
	}

	return []string{}
}

func features(doc *goquery.Document) map[string]string {
	results := map[string]string{}
	var key string

	doc.Find("table.product-detail__spec tr td").Each(func(i int, s *goquery.Selection) {
		if i%2 == 0 {
			key = s.Text()
		} else {
			a := s.Find("a")
			aText := strings.TrimSpace(a.Text())
			if aText != "" {
				results[key] = a.Text()
			} else {
				results[key] = strings.TrimSpace(s.Text())
			}
		}
	})

	return results
}

func images(doc *goquery.Document) []string {
	images := []string{}

	doc.Find("div[data-fancybox=\"gallery\"]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			images = append(images, href)
		}
	})

	return images
}

func analogPaths(doc *goquery.Document) []string {
	paths := []string{}

	doc.Find("div#analogues a.product-card__image").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			paths = append(paths, href)
		}
	})

	return paths
}

func attributes(doc *goquery.Document) []*attribute {
	results := []*attribute{}

	attrs := doc.Find(".product-detail-description-content__item")

	attrs.Each(func(i int, s *goquery.Selection) {
		text, err := s.Find(".product-detail-description-content__item-content div").Html()
		if err != nil {
			log.Fatal(err)
		}
		results = append(
			results,
			&attribute{
				Name:          strings.TrimSpace(s.Find("h3").First().Text()),
				SubAttributes: attrValues(strings.TrimSpace(text)),
			},
		)
	})

	return results[:len(results)-2]
}

func attrValues(text string) []*subAttribute {
	result := []*subAttribute{}

	blocks := strings.Split(text, "<br/><b>")

	for _, b := range blocks {
		var name string
		var vals []string
		var preVals []string

		if strings.Contains(b, "</b>") {
			re := regexp.MustCompile(`(.+)</b><br/>(.+)`)
			submatch := re.FindStringSubmatch(b)
			if len(submatch) == 3 {
				name = submatch[1]
				preVals = strings.Split(submatch[2], "<br/><br/>")
			}
		} else {
			preVals = strings.Split(b, "<br/><br/>")
		}

		for _, v := range preVals {
			vals = append(vals, strings.Split(v, "<br/>")...)
		}

		n := 0
		for _, v := range vals {
			trimmedV := strings.TrimSpace(v)
			if trimmedV != "" {
				vals[n] = trimmedV
				n++
			}
		}

		if n != 0 {
			result = append(result, &subAttribute{Name: name, Values: vals[:n]})
		}
	}

	return result
}

func extract(meds chan *medicament, quit chan bool, limit int, ticker *time.Ticker) {
	mapped := map[string]bool{}

	for _, letP := range letterPaths(newDoc(PATH_MNN, nil, ticker)) {
		let := letP[len(letP)-1:]
		for _, mnnP := range mnnPaths(newDoc(PATH_MNN, map[string]string{"abc": let}, ticker)) {
			for _, medP := range medicamentPaths(newDoc(mnnP, nil, ticker), ticker, true) {
				if isIn(medP, mapped) {
					continue
				}

				meds <- newMedicament(medP, newDoc(medP, nil, ticker))
				mapped[medP] = true

				if len(mapped) == limit {
					quit <- true
					return
				}
			}
		}
	}

	quit <- true
}

func newDoc(path string, queryParams map[string]string, ticker *time.Ticker) *goquery.Document {
	req, err := http.NewRequest("GET", URL+path, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")

	if queryParams != nil {
		q := req.URL.Query()
		for key, val := range queryParams {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatal(fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status))
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	<-ticker.C

	return doc
}

func isIn(s string, m map[string]bool) bool {
	_, ok := m[s]
	return ok
}

func insertAll(collectionName string, meds chan *medicament, quit chan bool) {
	mongoClient := mongodb.NewMongoClient()
	for {
		select {
		case m := <-meds:
			mongoClient.InsertOne(collectionName, m)
		case <-quit:
			return
		}
	}
}

func ParseAll(collectionName string) {
	meds := make(chan *medicament, 1)
	quit := make(chan bool)
	ticker := time.NewTicker(time.Second)

	go insertAll(collectionName, meds, quit)

	extract(meds, quit, -1, ticker)

	println(prettyPrint(meds))
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}
