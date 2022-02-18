# Broom [![Build](https://github.com/bojanz/broom/actions/workflows/build.yml/badge.svg)](https://github.com/bojanz/broom/actions/workflows/build.yml)

Broom is an API client powered by OpenAPI.

Point it to an OpenAPI spec, and it will provide a CLI/TUI for each defined operation.
JSON output is colored and formatted, authentication is handled.

## Install

```bash
go install github.com/bojanz/broom/cmd/broom@latest
```
or run `make` yourself inside the source directory, then copy the binary from bin/.

## Usage

```bash
cd my-project/

# Generate a .broom.yaml with an "api" profile.
# See broom add --help for more examples.
broom add api openapi.yaml

# Run "broom" without arguments to get a list of profiles.
broom

# Specify a profile to get a list of operations.
broom api

# Run a specific operation.
broom api list-products

# Show the response code and headers.
broom api list-products -v

# Headers are passed via -H.
broom api list-products -H "X-MyHeader: Value" -H "X-Another: Value2"

# Optional parameters are passed via -q.
broom api list-products -q "filter[owner_id]=my-user&sort=-sku"

# Required parameters are passed directly.
broom api get-product 01FAZ7A1H11FW16WPQZP879YX3

# Request body parameters are passed via -b.
# The query string is auto-mapped to JSON if the service requires it.
broom api create-product -b "name=T-Shirt&price=999&currency_code=EUR"

# Omitting -b will open a terminal UI for providing body parameters.
broom api create-product

# Get the list of all arguments and parameters via --help.
broom api create-product --help
```

## Profiles

Broom allows creating multiple profiles for working with different environments, e.g. staging and production.
Each profile has its own server url and authentication settings.

The auth type, API key header, and server url are auto-detected from the OpenAPI spec, when not provided via options.

```bash
cd my-project/
broom add prod openapi.json --auth=PRODUCTION_KEY
broom add staging openapi.json --auth=STAGING_KEY --server-url=htts://staging.my-api.io

# Proceed as usual.
broom prod list-products
broom staging list-products
```

## Authentication

Broom supports authenticating using an API key, Basic auth, or a Bearer token.

Using an API key (X-API-Key header):
```
broom add api openapi.json --auth=MYKEY --auth-type=api-key
```

Using an API key (custom header):
```
broom add api openapi.json --auth=MYKEY --auth-type=api-key --api-key-header="X-MyApp-Key"
```

Using Basic auth:
```
broom add api openapi.json --auth="username:password" --auth-type=basic
```

Using a Bearer token:
```
broom add api openapi.json --auth=MYKEY --auth-type=bearer
```

For more advanced use cases, Broom supports fetching credentials through an external command:
```
    broom add api openapi.json --auth-cmd="sh get-token.sh" --auth-type=bearer
```

The external command can do a 2-legged OAuth request via curl, or it can retrieve an API key from a vault.
It is run before each request to ensure freshness.

## Name

Named after a curling broom, with bonus points for resembling the sound a car makes (in certain languages).

## Alternatives

[Restish](https://rest.sh) does the non-TUI part of this tool with many additional features.
