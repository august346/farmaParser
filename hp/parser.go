package hp

import (
	"farma/parser"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	HREF_LETTERS string = "/ingredients/"
)

var URL string

type subAttribute struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type attribute struct {
	Name          string          `json:"name"`
	SubAttributes []*subAttribute `json:"subAttribute"`
}

type medicament struct {
	Href       string            `json:"href"`
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

func medicamentHrefs(f *parser.FarmaParser, href string, withNexts bool) []string {
	medsHrefsDoc := doc(f, href, nil)

	result := scrabHrefs("div.card-list__element a.product-card__image", medsHrefsDoc)

	if !withNexts {
		return result
	}

	moreMedsHrefs := scrabHrefs("div.pagination.pagination_large a.pagination__item", medsHrefsDoc)
	for _, moreMedsHref := range moreMedsHrefs {
		result = append(result, medicamentHrefs(f, moreMedsHref, false)...)
	}

	return result
}

func newMedicament(href string, doc *goquery.Document) *medicament {
	med := &medicament{
		Href:       href,
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

func Jobber(f *parser.FarmaParser) {
	URL = os.Getenv("HP_URL")

	gottenMedicaments := map[string]bool{}

	lettersHrefs := scrabHrefs("li.main-alphabet__nav-item a", doc(f, HREF_LETTERS, nil))
	for _, letterHref := range lettersHrefs {
		letter := letterHref[len(letterHref)-1:]
		mnnReqQuery := map[string]string{"abc": letter}
		letterDoc := doc(f, HREF_LETTERS, mnnReqQuery)

		mnnHrefs := scrabHrefs(".main-alphabet__list a", letterDoc)
		for _, mnnHref := range mnnHrefs {

			mnnMedsHrefs := medicamentHrefs(f, mnnHref, true)
			for _, medHref := range mnnMedsHrefs {
				if isIn(medHref, gottenMedicaments) {
					continue
				}

				medicamentDoc := doc(f, medHref, nil)
				medicament := newMedicament(medHref, medicamentDoc)
				f.RawMedicaments <- medicament

				gottenMedicaments[medHref] = true
			}
		}
	}
}

func doc(f *parser.FarmaParser, href string, queryParams map[string]string) *goquery.Document {
	req, err := http.NewRequest("GET", URL+href, nil)
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
