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

	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

func setUpProxy() {
	proxyURL := os.Getenv("PROXY_URL")
	proxyUsername := os.Getenv("PROXY_USERNAME")
	proxyPass := os.Getenv("PROXY_PASS")

	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	dialer, err := proxy.SOCKS5(
		"tcp",
		proxyURL,
		&proxy.Auth{User: proxyUsername, Password: proxyPass},
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
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
