# 0005. Go 1.22 stdlib `http.ServeMux` routing, no router package

## Context
The API needs method-aware, path-parameter routes like `PUT /api/boards/{id}` and `GET /api/tasks/{id}/attachments/{aid}`. Prior to Go 1.22, `net/http`'s `ServeMux` couldn't match on HTTP method or extract path parameters, so stdlib-only Go projects typically hand-roll a small router or path-parsing layer for this.

## Decision
Since the API already runs in a `golang:1.22` container (see [0001](0001-docker-only-dev-workflow.md)), `cmd/api/main.go` uses Go 1.22's enhanced `http.ServeMux` directly: patterns like `mux.HandleFunc("PUT /api/boards/{id}", ...)`, with `r.PathValue("id")` inside handlers to read path parameters. This satisfies "stdlib only" cleanly with zero extra code.

## Alternatives considered
- **Third-party router (chi, gorilla/mux, httprouter, etc.):** more features, but an external dependency the spec disallows.
- **Hand-rolled path parser on top of Go <1.22's `ServeMux`:** works without pinning a newer Go version, but is extra code to write and maintain for something the stdlib now does natively.

## Consequences
- The API's minimum Go version is a hard requirement (1.22), enforced implicitly by the `golang:1.22` base image and explicitly by `go.mod`'s `go 1.22` directive — building with an older toolchain fails immediately.
- Route registration in `main.go` stays flat and readable — no router-specific DSL or middleware-chaining syntax to learn.
