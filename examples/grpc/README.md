# grpc client example

Example gRPC client using [IAP authentication](https://cloud.google.com/iap/docs/authentication-howto).

## Description

The client here is based on the [gRPC Go examples](https://github.com/grpc/grpc-go/tree/master/examples).
It uses [oauth credentials](https://godoc.org/google.golang.org/grpc/credentials/oauth) to authenticate
with the [Identity Aware Proxy](https://cloud.google.com/iap/) on every gRPC call.

Note: gRPC oatuh authentication requires TLS.

The client-id is the oauth client_id in the credentials associated with your IAP configuration. See https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_service_account