# Broom

Broom is an API client powered by OpenAPI.

Point it to an OpenAPI spec, and it will provide a CLI/TUI for each defined operation.
JSON output is colored and formatted, authentication is handled.

## Install

```bash
go install github.com/bojanz/broom/cmd/broom@latest
go install github.com/bojanz/broom/cmd/broom-init@latest
```
or run `make` yourself inside the source directory, then copy the two binaries from bin/.

## Usage

Broom can be initialized and used per-project, or globally.

```bash
cd my-project/

# Generate a .broom.yaml with an "api" profile.
# See broom-init --help for more examples.
broom-init api openapi.yaml

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

The broom-config tool allows creating multiple profiles, to allow
working with multiple environments, e.g. staging and production.
Each profile has its own server url and authentication settings.

```bash
cd my-project/
broom-init prod openapi.json --token=PRODUCTION_KEY
broom-init staging openapi.json --token=STAGING_KEY --server-url=htts://staging.my-api.io

# Proceed as usual.
broom prod list-products
broom staging list-products
```

## Authentication

An access token can be set on the profile (via `broom init --token`) or provided per-operation (via `broom --token`).
This is the usual way of sending API keys.

For more advanced use cases, Broom supports fetching an access token through an external command:
```
    broom-init api openapi.json --token-cmd="sh get-token.sh"
```

The external command can do a 2-legged OAuth request via curl, or it can retrieve an API key from a vault.
It is run before each request to ensure freshness.

Note: Access tokens are currently always sent in an "Authorization: Bearer" header. The OpenAPI spec allows
specifying which header to use (e.g. "X-API-Key"), Broom should support that at some point.

## Name

Named after a curling broom, with bonus points for resembling the sound a car makes (in certain languages).

## Alternatives

[Restish](https://rest.sh) does the non-TUI part of this tool with many additional features.
