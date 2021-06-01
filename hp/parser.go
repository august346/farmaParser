package hp

import (
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
	Path       string            `json:"path"`
	Groups     []string          `json:"groups"`
	Title      string            `json:"title"`
	Price      float32           `json:"price"`
	Images     []string          `json:"images"`
	Features   map[string]string `json:"features"`
	Attributes []*attribute      `json:"attributes"`
}

func scrabHrefs(selector string, doc *goquery.Document) []string {
	result := []string{}

	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			result = append(result, href)
		}
	})

	return result
}

func medicamentPaths(doc *goquery.Document, ticker *time.Ticker, withNexts bool) []string {
	result := scrabHrefs("div.card-list__element a.product-card__image", doc)

	if !withNexts {
		return result
	}

	for _, nextLink := range scrabHrefs("div.pagination.pagination_large a.pagination__item", doc) {
		nextDoc := newDoc(nextLink, nil, ticker)
		result = append(result, medicamentPaths(nextDoc, ticker, false)...)
	}

	return result
}

func newMedicament(path string, doc *goquery.Document) *medicament {
	med := &medicament{
		Path:       path,
		Title:      strings.TrimSpace(doc.Find("h1.product-detail__title").Text()),
		Groups:     groups(doc),
		Images:     scrabHrefs("div[data-fancybox=\"gallery\"]", doc),
		Features:   features(doc),
		Attributes: attributes(doc),
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

	for _, letP := range scrabHrefs("li.main-alphabet__nav-item a", newDoc(PATH_MNN, nil, ticker)) {
		let := letP[len(letP)-1:]
		mnnReqQuery := map[string]string{"abc": let}
		letDoc := newDoc(PATH_MNN, mnnReqQuery, ticker)
		for _, mnnP := range scrabHrefs(".main-alphabet__list a", letDoc) {
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

	res, err := http.DefaultClient.Do(req)
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

	println(path)

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
}
