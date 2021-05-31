package oz

import (
	"bytes"
	"encoding/json"
	"farma/jq"
	"farma/mongodb"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

var URL string = string([]rune{104, 116, 116, 112, 115, 58, 47, 47, 119, 119, 119, 46, 114, 105, 103, 108, 97, 46, 114, 117, 47, 103, 114, 97, 112, 104, 113, 108})

type requestJson struct {
	Query     string         `json:"query"`
	Variables map[string]int `json:"variables"`
}

type querries struct {
	Request   string
	Transform string
}

type parser struct {
	HTTPClient *http.Client
	Querries   querries
	PageSize   int
}

func requstJsonToBytes(r requestJson) []byte {
	value, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}
	return value
}

func newParser(pageSize int) *parser {
	graphql, err := ioutil.ReadFile("files/oz.graphql")
	if err != nil {
		log.Fatal(err)
	}

	jq, err := ioutil.ReadFile("files/oz.jq")
	if err != nil {
		log.Fatal(err)
	}

	return &parser{
		HTTPClient: http.DefaultClient,
		Querries: querries{
			Request:   string(graphql),
			Transform: string(jq),
		},
		PageSize: pageSize,
	}
}

func (p *parser) requestJSONBytes(pageNumber int) []byte {
	return requstJsonToBytes(
		requestJson{
			Query: p.Querries.Request,
			Variables: map[string]int{
				"page": pageNumber,
				"size": p.PageSize,
			},
		},
	)
}

func (p *parser) request(requestJSONBytes []byte) *http.Request {
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(requestJSONBytes))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")
	req.Header.Set("Content-Type", "application/json")

	return req
}

func (p *parser) responseBody(request *http.Request) []byte {
	resp, err := p.HTTPClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return body
}

func (p *parser) transformedJSON(responseBody []byte) []interface{} {
	input := map[string]interface{}{
		"response_body": string(responseBody),
	}

	transformed, err := jq.Transform(
		input,
		string(p.Querries.Transform),
	)
	if err != nil {
		log.Fatal(err)
	}

	return transformed.([]interface{})
}

func (p *parser) items(pageNumber int) []interface{} {
	reqJSONBytes := p.requestJSONBytes(pageNumber)

	req := p.request(reqJSONBytes)

	body := p.responseBody(req)

	return p.transformedJSON(body)
}

func ParseAll(collectionName string, limit int) {
	mongoClient := mongodb.NewMongoClient()

	p := newParser(limit)

	for i := 0; ; i++ {
		items := p.items(i)
		if len(items) == 0 {
			return
		}

		mongoClient.InsertMany(collectionName, items)

		time.Sleep(time.Second)
		println("Parsed", i+1)
	}
}
