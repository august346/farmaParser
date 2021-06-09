package oz

import (
	"bytes"
	"encoding/json"
	"farma/jq"
	"farma/parser"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var URL string

type requestJson struct {
	Query     string         `json:"query"`
	Variables map[string]int `json:"variables"`
}

func Jobber(f *parser.FarmaParser) {
	var err error
	var tmpBytes []byte
	var graphqlQuery string
	var jqQuery string

	URL = os.Getenv("OZ_URL")

	tmpBytes, err = ioutil.ReadFile("files/oz.graphql")
	if err != nil {
		log.Fatal(err)
	}
	graphqlQuery = string(tmpBytes)

	tmpBytes, err = ioutil.ReadFile("files/oz.jq")
	if err != nil {
		log.Fatal(err)
	}
	jqQuery = string(tmpBytes)

	for i := 0; ; i++ {
		f.Jobs <- &parser.ResponseJob{
			Type:    "bytes",
			Request: request(graphqlQuery, i, 20),
		}

		rspBytes := <-f.RspBytes
		if rspBytes.Err != nil {
			log.Fatal(err)
		}

		transformed, err := jq.Transform(
			map[string]interface{}{
				"response_body": string(rspBytes.Bytes),
			},
			string(jqQuery),
		)
		if err != nil {
			log.Fatal(err)
		}

		for _, rawMed := range transformed.([]interface{}) {
			f.RawMedicaments <- rawMed
		}
	}
}

func request(query string, pageNumber int, pSize int) *http.Request {
	reqBodyObject := &requestJson{
		Query: query,
		Variables: map[string]int{
			"page": pageNumber,
			"size": pSize,
		},
	}
	reqBody, err := json.Marshal(reqBodyObject)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")
	req.Header.Set("Content-Type", "application/json")

	return req
}
