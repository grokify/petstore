# TRD — Petstore server with optional AAuth

**Initiative:** `INIT-PETSTORE-001`
**Repository:** `github.com/grokify/petstore`

## Layout

| Path | Contents |
|---|---|
| `petstore.go` | Package `petstore`: the embedded `openapi.yaml` |
| `server/` | Package `server`: store, handlers, spec-driven router |
| `cmd/petstore-server/` | Cobra command over `server` |

## Spec annotations

Each operation carries `x-aauth`:

```yaml
x-aauth:
  access: auth        # agent | person | auth
  scope: write:pets   # required when access is auth
```

The top level carries `x-aauth.scopes`, the descriptions published as the
resource metadata's `scope_descriptions` for consent screens.

## Routing

`server.New` reads the spec's operations (method, path, `operationId`,
`x-aauth`) and the base path of its first server URL (`/v2`). It registers
each operation on an `http.ServeMux` as `METHOD /v2/path`; OpenAPI path
templates (`{petId}`) are ServeMux wildcards. With a guard, each handler is
wrapped by `ResourceGuard.Require` (aauth-go), and the guard's `WellKnown`
handler serves `/.well-known/aauth-resource.json` and `/jwks.json`.

`GET /openapi.yaml` and `GET /openapi.json` serve the spec with `servers`
replaced by the server's public URL and base path.

## AAuth mode

| Flag | Meaning |
|---|---|
| `--auth aauth` | enable AAuth |
| `--url` | the resource's identifier, such as `https://petstore.aauth.test` |
| `--trust` | person server issuers to accept (repeatable) |
| `--ca-file` | extra root CA for fetching issuers' keys (the sandbox CA) |

The resource signing key is generated at start; its public key is served
at `/jwks.json`, so restarts need no key file.

## Store

A mutex-guarded in-memory store: pets by ID, orders by ID with the owner's
`(iss, sub)`, users by username. Passwords are accepted and never returned.
