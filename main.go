package main

import (
	"errors"
	"farma/gz"
	"farma/hp"
	"farma/oz"
	"farma/parser"
	"fmt"
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

func main() {
	var ticker *time.Ticker
	var jobber func(f *parser.FarmaParser)

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	setUpProxy()

	args := os.Args[1:]

	switch args[0] {
	case "oz":
		ticker = time.NewTicker(2 * time.Second)
		jobber = oz.Jobber
	case "gz":
		ticker = time.NewTicker(time.Second)
		jobber = gz.Jobber
	case "hp":
		ticker = time.NewTicker(time.Second)
		jobber = hp.Jobber
	default:
		log.Fatalf("Unknown jobber `%s`!", args[0])
	}

	fmt.Printf("started `%s`\n", args[0])

	parser.NewRawFarmaParser(ticker, args[1]).Run(jobber)

	fmt.Println("parsed")

	fmt.Println("ended")
}
