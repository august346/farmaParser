package parser

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	Paths chan string
	Docs  chan *Doc
	Pause time.Duration
	URL   string
}

type Doc struct {
	Path string
	Doc  *goquery.Document
}

func (p *Parser) Parse() {
	var c int
	limiter := time.NewTicker(p.Pause)

	for {
		pth, ok := <-p.Paths
		if !ok {
			return
		}

		<-limiter.C

		fmt.Println(c, time.Now().Format("2006-01-02T15:04:05.000Z"), pth)
		c++

		p.Docs <- &Doc{
			Path: pth,
			Doc:  htmlDocument(Link(p.URL, pth)),
		}
	}
}

func Link(URL string, pth string) string {
	u, err := url.Parse(URL)
	if err != nil {
		log.Fatal(err)
	}

	u.Path = path.Join(u.Path, pth)
	return u.String()
}

func htmlDocument(link string) *goquery.Document {
	client := &http.Client{}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")

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

	return doc
}
