# Petstore

`petstore` is an example app that extends the Swagger Petstore API definition
(`openapi.yaml`) with a server, `petstore-server`, that serves every operation
from memory. Optionally, the server is an [AAuth](https://github.com/aauth-dev/aauth-go)
resource: each operation asks an AI agent for exactly the trust it needs.

## Run the Server

```bash
go run ./cmd/petstore-server --listen 127.0.0.1:8080
curl 'http://127.0.0.1:8080/v2/pet/findByStatus?status=available'
```

The store starts with six pets and keeps everything in memory. The spec is
served at `/openapi.yaml` and `/openapi.json`, with `servers` pointing at the
server, so tools that load an OpenAPI document find it there.

## Run It as an AAuth Resource

With `--auth aauth`, each operation requires what its `x-aauth` annotation in
`openapi.yaml` says:

| Operations | Requires | What the agent does |
|---|---|---|
| `findPetsByStatus`, `findPetsByTags`, `getPetById`, `getInventory`, `loginUser`, `logoutUser` | the agent's identity | signs the request with its agent token |
| `getOrderById`, user operations | the person | presents a person token from its person server |
| `placeOrder`, `deleteOrder` | an auth token for `write:orders` | asks the person to approve, then presents the auth token |
| `addPet`, `updatePet`, `updatePetWithForm`, `deletePet`, `uploadFile` | an auth token for `write:pets` | the same, for `write:pets` |

Orders belong to the person who placed them: another person's order is not
found.

The quickest way to try it is behind the
[aauth-go sandbox](https://aauth-dev.github.io/aauth-go/guides/sandbox/),
which runs an agent provider and a person server and serves the Petstore as
`https://petstore.aauth.test`:

```bash
aauth-sandbox init
petstore-server --listen 127.0.0.1:8080 --auth aauth \
  --url https://petstore.aauth.test \
  --trust https://ps.aauth.test \
  --ca-file ~/.aauth-sandbox/ca.pem
aauth-sandbox serve --upstream petstore=http://127.0.0.1:8080
```

| Flag | Meaning |
|---|---|
| `--auth` | `none` (default) or `aauth` |
| `--url` | the resource's public URL and AAuth identifier |
| `--trust` | a person server whose tokens to accept (repeatable) |
| `--ca-file` | an extra root CA for fetching issuers' keys, such as the sandbox CA |
| `--listen` | the address to listen on, plain HTTP (default `:8080`) |

The resource metadata is at `/.well-known/aauth-resource.json`, with the scope
descriptions a person server shows on its consent page.

To test agents and AAuth implementations against the Petstore, see the
[interop guide](docs/guides/aauth-interop.md).

## Container

```bash
docker build -t petstore-server .
docker run --rm -p 127.0.0.1:8080:8080 petstore-server
```
