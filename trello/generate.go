// Package trello / generate.go: standalone entrypoint for ``go generate``.
//
// Running ``go generate ./...`` from the repo root will (re)build the SDK
// from the latest upstream OpenAPI spec by invoking scripts/generate.sh.
package trello

//go:generate bash ../scripts/generate.sh
