# HTTP client example

Example HTTP request using [IAP authentication](https://cloud.google.com/iap/docs/authentication-howto).

## Description

This example shows manually using IAP tokens to authenticate an HTTP request.

One could also use an [oauth2 client](https://godoc.org/golang.org/x/oauth2#NewClient):

```go
// error handling ommitted
package main

import (
    "golang.org/x/oauth2"
    iap "github.com/bakins/iap-token-source"
)

func main() {
    t, err := iap.New(MY-CLIENT-ID)
    c := oauth2.NewClient(context.Background(), t)
    // now use c as normal
}
```
 
The client-id is the oauth client_id in the credentials associated with your IAP configuration. See https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_service_account