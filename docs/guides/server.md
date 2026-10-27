# Run the Server

`petstore-server` serves every operation in the Petstore OpenAPI definition
(`openapi.yaml`, OpenAPI 3.1) from memory. It can also act as an
[AAuth](https://github.com/aauth-dev/aauth-go) resource, so each operation asks
an AI agent for exactly the trust it needs.

## Run It

```bash
go run ./cmd/petstore-server --listen 127.0.0.1:8080
curl 'http://127.0.0.1:8080/v2/pet/findByStatus?status=available'
```

- **Operations.** All 20 operations of the spec are routed under the spec's base
  path, `/v2`: the pet, store and user groups.
- **Store.** It starts with six pets and keeps everything in memory, so a pet
  that an order sells is gone, and anything you add is lost, when the server
  restarts.
- **Spec.** The spec is served at `/openapi.yaml` and `/openapi.json`, with
  `servers` pointing at the server, so tools that load an OpenAPI document find
  it there. The server embeds the spec; it is also importable as the Go
  package `github.com/grokify/petstore`.

## Options

| Flag | Meaning |
|------|---------|
| `--listen` | the address to listen on, plain HTTP (default `:8080`); put TLS in front, as the sandbox does |
| `--auth` | `none` (default) serves every operation to anyone; `aauth` makes the server an AAuth resource |
| `--url` | the resource's public URL and AAuth identifier, such as `https://petstore.aauth.test` |
| `--trust` | a person server whose tokens to accept (repeatable; required with `--auth aauth`) |
| `--ca-file` | a PEM file of an extra root CA for fetching issuers' keys, such as the sandbox CA |

## Run It as an AAuth Resource

With `--auth aauth`, each operation requires what its `x-aauth` annotation in
`openapi.yaml` says, from the agent's identity up to the person's approval of a
scope. The levels, and how to run the server next to the AAuth sandbox and check
an agent against it, are in [Test AAuth interop](aauth-interop.md).

The resource metadata is at `/.well-known/aauth-resource.json`, with the scope
descriptions a person server shows on its consent page.

## Container

```bash
docker build -t petstore-server .
docker run --rm -p 127.0.0.1:8080:8080 petstore-server
```

The image is static and runs as a non-root user. It has no shell or curl, so
its health check is the server's own `health` subcommand, which asks a running
server for its spec:

```bash
petstore-server health --addr 127.0.0.1:8080
```

Publish the port on `127.0.0.1` as above unless you mean to expose an
unauthenticated server: without `--auth aauth` anyone who can reach it can
change its pets.
