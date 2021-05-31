package main

import (
	"encoding/json"
	"errors"
	"farma/gz"
	"farma/hp"
	"farma/oz"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

func setUpProxy() {
	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	dialer, err := proxy.SOCKS5(
		"tcp",
		PROXY_URL,
		&proxy.Auth{User: PROXY_USERNAME, Password: PROXY_PASS},
		baseDialer,
	)
	if err != nil {
		log.Fatal(err)
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		log.Fatal(errors.New("fails contextDialer init"))
	}

	httpTransport := &http.Transport{}
	http.DefaultClient.Transport = httpTransport
	httpTransport.DialContext = contextDialer.DialContext
}

type checkIPResult struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	CC      string `json:"cc"`
}

func checkIP() {
	resp, err := http.Get("https://api.myip.com/")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var chRes checkIPResult
	err = json.NewDecoder(resp.Body).Decode(&chRes)
	if err != nil {
		log.Fatal(err)
	} else if chRes.CC == "RU" {
		log.Fatal(errors.New("proxy broken"))
	}

	log.Printf("%q \n", chRes)
}

func main() {
	setUpProxy()
	checkIP()

	println("started")

	args := os.Args[1:]

	switch args[0] {
	case "oz":
		oz.ParseAll(args[1], 10)
	case "gz":
		gz.ParseAll(args[1], args[2])
	case "hp":
		hp.ParseAll(args[1])
	}

	println("parsed")

	println("ended")
}
