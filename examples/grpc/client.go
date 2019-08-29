/*
 * based on https://github.com/grpc/grpc-go/blob/master/examples/helloworld/greeter_client/main.go
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"time"

	iap "github.com/bakins/iap-token-source"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 3 {
		log.Fatalf("usage %s CLIENT_ID ADDRESS", os.Args[0])
	}

	audience := os.Args[1]
	address := os.Args[2]

	iap, err := iap.New(context.Background(), audience, "")
	if err != nil {
		log.Fatalf("failed to create IAP token source: %v", err)
	}

	// change InsecureSkipVerify to true if you want to skip verifying the server certificate.
	// Useful when using a self signed certificate.
	// oauth authentication with gRPC MUST use TLS
	t := credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(t),
		// add an authorization token from the IAP token source to every gRPC client call
		grpc.WithDefaultCallOptions(grpc.PerRPCCredentials(oauth.TokenSource{TokenSource: iap})),
	}

	conn, err := grpc.Dial(address, options...)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}

	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: "world"})
	if err != nil {
		log.Fatalf("SayHello failed: %v", err)
	}
	log.Printf("Server said: %s", r.Message)
}
