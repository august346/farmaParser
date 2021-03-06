package parser

import (
	"encoding/json"
	"farma/jq"
	"farma/mongodb"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const URL_CHECK_IP string = "https://api.myip.com/"

type image struct {
	Thumbnail string
	Main      string
}

type generalInfo struct {
	MNN          string
	Manufacturer string
	Doze         string
	Numero       string
	Form         string
}

type medicament struct {
	Title       string
	URL         string
	Price       float32
	Images      []*image
	Groups      []string
	GeneralInfo *generalInfo
	Description map[string]string
	Attributes  map[string]string
}

type ResponseJob struct {
	Type    string
	Request *http.Request
}

type RspDoc struct {
	Doc *goquery.Document
	Err error
}

type RspByte struct {
	Bytes []byte
	Err   error
}

type FarmaParser struct {
	ticker         *time.Ticker
	RspDocs        chan *RspDoc
	RspBytes       chan *RspByte
	Jobs           chan *ResponseJob
	RawMedicaments chan interface{}
	instructionsJQ string
	mongoClient    *mongodb.MongoClient
	needTransform  bool
}

func NewRawFarmaParser(ticker *time.Ticker, collectionName string) *FarmaParser {
	mClient := mongodb.NewMongoClient()
	mClient.CollectionName = collectionName

	return &FarmaParser{
		ticker:         ticker,
		RspDocs:        make(chan *RspDoc),
		RspBytes:       make(chan *RspByte),
		Jobs:           make(chan *ResponseJob),
		RawMedicaments: make(chan interface{}),
		mongoClient:    mClient,
		needTransform:  false,
	}
}

type checkProxyResult struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	CC      string `json:"cc"`
}

func (f *FarmaParser) response(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	} else if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad response status code: %d", resp.StatusCode)
	}

	return resp, nil
}

func (f *FarmaParser) responseBytes(r *http.Request) ([]byte, error) {
	resp, err := f.response(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (f *FarmaParser) responseDoc(r *http.Request) (*goquery.Document, error) {
	resp, err := f.response(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (f *FarmaParser) runParse() {
	var c int

	for {
		job := <-f.Jobs

		switch job.Type {
		case "doc":
			doc, err := f.responseDoc(job.Request)
			f.RspDocs <- &RspDoc{doc, err}
		case "bytes":
			bytes, err := f.responseBytes(job.Request)
			f.RspBytes <- &RspByte{bytes, err}
		default:
			log.Fatal(fmt.Sprintf("unknown job type `%s`", job.Type))
		}

		fmt.Println(c, time.Now().Format("2006-01-02T15:04:05.000Z"))
		c++

		<-f.ticker.C
	}
}

func (f *FarmaParser) checkProxy() *checkProxyResult {
	resp, err := http.DefaultClient.Get(URL_CHECK_IP)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var cpr *checkProxyResult
	err = json.NewDecoder(resp.Body).Decode(&cpr)
	if err != nil {
		log.Fatal(err)
	} else if cpr.CC == "RU" {
		log.Fatal(fmt.Sprintf("broken proxy: check IP result - %q", cpr))
	}

	return cpr
}

func (f *FarmaParser) transformJSON(data interface{}) (*medicament, error) {
	transformed, err := jq.Transform(data, f.instructionsJQ)
	if err != nil {
		return nil, err
	}

	return transformed.(*medicament), nil
}

func (f *FarmaParser) runInsertions() {
	var data interface{}
	var err error

	for {
		data = <-f.RawMedicaments

		if f.needTransform {
			data, err = f.transformJSON(data)
			if err != nil {
				log.Fatal(err)
			}
		}
		f.mongoClient.Insert(data)
	}
}

func (fp *FarmaParser) Run(f func(*FarmaParser)) {
	fmt.Printf("Proxy OK: %q\n", *fp.checkProxy())

	go fp.runParse()
	go fp.runInsertions()

	f(fp)
}
