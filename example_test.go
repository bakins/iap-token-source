package iap

import (
	"context"
	"log"
	"os"

	"golang.org/x/oauth2"
)

func ExampleNew() {
	audience := os.Getenv("AUDIENCE")
	t, err := New(context.Background(), audience, "")
	if err != nil {
		log.Fatalf("failed to create token source: %v", err)
	}

	c := oauth2.NewClient(context.Background(), t)
	_, _ = c.Get("https://my-iap.protected.service")
}
