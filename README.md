# go-trello-sdk

A Go SDK for the [Trello REST API], generated from the official
[OpenAPI 3.0 specification][spec] using [oapi-codegen].

[Trello REST API]: https://developer.atlassian.com/cloud/trello/rest/
[spec]: https://dac-static.atlassian.com/cloud/trello/swagger.v3.json
[oapi-codegen]: https://github.com/oapi-codegen/oapi-codegen

## Installation

```bash
go get github.com/lonegunmanb/go-trello-sdk/trello
```

## Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/lonegunmanb/go-trello-sdk/trello"
)

func main() {
    c, err := trello.New(
        trello.WithCredentials("YOUR_API_KEY", "YOUR_API_TOKEN"),
    )
    if err != nil {
        panic(err)
    }

    resp, err := c.GetMembersIdWithResponse(context.Background(), "me", &trello.GetMembersIdParams{})
    if err != nil {
        panic(err)
    }
    fmt.Printf("status=%d body=%s\n", resp.StatusCode(), string(resp.Body))
}
```

Generate Trello [API key & token](https://trello.com/power-ups/admin) from your
own Trello account.

### Available options

| Option                       | Purpose                                                                |
|------------------------------|------------------------------------------------------------------------|
| `WithServer(url)`            | Override the API base URL (defaults to `https://api.trello.com/1`).    |
| `WithCredentials(key, tok)`  | Append `?key=…&token=…` to every request.                              |
| `WithHTTPDoer(doer)`         | Plug in a custom `*http.Client` (e.g. with timeouts, retries, proxy).  |
| `WithRequestEditor(fn)`      | Append an arbitrary request mutator (runs after the credentials editor). |

### Method names

Method names follow oapi-codegen's convention `<Verb><PathInPascalCase>` and
mirror the operationIds in the OpenAPI spec, for example:

| Trello endpoint                       | Generated method                              |
|---------------------------------------|------------------------------------------------|
| `GET /members/{id}`                   | `GetMembersIdWithResponse`                     |
| `GET /members/{id}/boards`            | `GetMembersIdBoardsWithResponse`               |
| `GET /boards/{id}`                    | `GetBoardsIdWithResponse`                      |
| `GET /boards/{id}/lists`              | `GetBoardsIdListsWithResponse`                 |
| `GET /boards/{id}/cards`              | `GetBoardsIdCardsWithResponse`                 |

The `*WithResponse` helpers return a typed response with `Body []byte`,
`HTTPResponse *http.Response`, and `StatusCode()` / `Status()` accessors. For
endpoints whose response schema is a `oneOf` union (which is most of the
Trello API), unmarshal `resp.Body` into your own struct as shown above.

If you'd rather work with raw `*http.Response`, every endpoint also has a
non-`WithResponse` variant on `*Client`.

## Repository layout

```
.
├── api/                       # Spec + generator config
│   ├── trello-swagger.json    # Vendored upstream OpenAPI 3.0 spec
│   ├── oapi-codegen.yaml      # oapi-codegen configuration
│   └── preprocess_spec.py     # Workarounds for upstream spec bugs
├── trello/                    # The SDK
│   ├── trello.gen.go          # Generated client + models (do not edit)
│   ├── client.go              # Hand-written constructor & options
│   ├── client_test.go         # Hand-written unit tests
│   └── generate.go            # `go generate` entry point
├── acceptance/                # Read-only acceptance tests
│   └── acceptance_test.go     # Live API calls gated by env vars
├── scripts/
│   └── generate.sh            # Regenerates trello/trello.gen.go
└── .github/workflows/ci.yml   # Build + unit + acceptance CI
```

## Regenerating the SDK

The generated client is committed to the repository so consumers can
`go get` it without having to run code generation themselves. To pick up
upstream Trello API changes:

```bash
./scripts/generate.sh         # downloads the spec and regenerates trello.gen.go
go test ./...                 # sanity-check the result
```

The script downloads the latest spec, runs `api/preprocess_spec.py` to patch a
small handful of upstream bugs (duplicate `operationId`s, an invalid
`number/integer` numeric format, a missing path parameter, and `oneOf` path
parameter schemas that confuse the generator), then invokes
`oapi-codegen v2.4.1`.

## Acceptance tests

The `acceptance` package exercises the SDK against the live Trello API using a
small set of **read-only** endpoints:

- `GET /members/me`
- `GET /members/me/boards`
- `GET /boards/{id}`
- `GET /boards/{id}/lists`

No write operations are invoked, so running these tests cannot modify any
existing Trello data.

The tests skip themselves when `TRELLO_API_KEY` or `TRELLO_API_TOKEN` are not
set, so contributors without credentials can still run `go test ./...`. In CI
they execute on every push and pull request from the main repository, using
the matching repository secrets.

To run them locally:

```bash
export TRELLO_API_KEY=...
export TRELLO_API_TOKEN=...
go test -v ./acceptance/...
```

## License

See [LICENSE](LICENSE) if/when added. The Trello REST API and OpenAPI spec
are © Atlassian; this project is an independent client and is not affiliated
with or endorsed by Atlassian.
