package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	iap "github.com/bakins/iap-token-source"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 3 {
		log.Fatalf("usage %s CLIENT_ID URL", os.Args[0])
	}

	audience := os.Args[1]
	url := os.Args[2]

	iap, err := iap.New(audience)
	if err != nil {
		log.Fatalf("failed to create IAP token source: %v", err)
	}

	// this example shows getting and using the token manually.
	// You could also use a client created using https://godoc.org/golang.org/x/oauth2#NewClient
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("NewRequest failed: %v", err)
	}

	token, err := iap.Token()
	if err != nil {
		log.Fatalf("failed to get token: %v", err)
	}

	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to read response body: %v", err)
	}

	log.Printf("HTTP Status: %d", resp.StatusCode)
	log.Print("Response Body:")
	log.Print(string(data))
}
