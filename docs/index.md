# OpenAPI Petstore

The **OpenAPI Petstore** is the standard Swagger/OpenAPI sample API
(OpenAPI **3.1.0**) with a working server, `petstore-server`, that serves every
operation from memory. The server can also act as an
[AAuth](https://github.com/aauth-dev/aauth-go) resource: each operation asks an
AI agent for exactly the trust it needs, from just its identity to the
person's approval of a scope.

That makes it a small, realistic target for testing agents and AAuth
implementations against each other.

## Start Here

- **[Test AAuth interop](guides/aauth-interop.md)** runs the Petstore next to
  the AAuth sandbox and checks an agent, a person server or your own
  implementation against it.
- **[Postman AI readiness](guides/postman/index.md)** scores how usable the API
  is by an AI agent, with the Postman CLI.
- The [README](https://github.com/grokify/petstore#readme) covers running the
  server, its flags and the container image.

## The Spec

The spec, `openapi.yaml`, is at the repository root. The server embeds it and
serves it at `/openapi.yaml` and `/openapi.json`. Each operation's `x-aauth`
annotation says what it requires in AAuth mode. The Postman smoke collection
and mock are under `postman/`.
