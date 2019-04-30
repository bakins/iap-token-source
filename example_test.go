package iap

import (
	"context"
	"log"
	"os"

	"golang.org/x/oauth2"
)

func ExampleNew() {
	clientID := os.Getenv("CLIENT_ID")
	t, err := New(clientID)
	if err != nil {
		log.Fatalf("failed to create token source: %v", err)
	}

	c := oauth2.NewClient(context.Background(), t)
	_, _ = c.Get("https://my-iap.protected.service")
}
