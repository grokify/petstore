# PRD — Petstore server with optional AAuth

**Initiative:** `INIT-PETSTORE-001`
**Repository:** `github.com/grokify/petstore`

## Problem

The Swagger Petstore is the API most developers already know, and this
repository carries its OpenAPI definition, but no server. Demos of agent
authorization need a resource that an agent can call for real: one with
operations that need different levels of trust, and a spec an agent can
load as a tool set.

AAuth (draft-hardt-oauth-aauth-protocol) lets a resource ask for exactly
what each operation needs: the agent's identity, the person the agent acts
for, or the person's consent for a scope. The Postman collection and the
omniagent demo should call the same resource, behind the same person
server and agent provider, so the two demos mirror each other.

## Goals

- `petstore-server` serves every operation in `openapi.yaml` from memory,
  seeded with pets, with no database.
- With `--auth aauth` it is an AAuth resource: each operation's access level
  comes from an `x-aauth` annotation in the spec, so the spec documents the
  policy the server enforces.
- Without it, it is a plain Petstore, for API demos that do not involve
  AAuth.
- It runs behind the aauth-go sandbox as `https://petstore.aauth.test`, and
  as a container.
- It serves its own spec, with `servers` pointing at itself, for agents that
  load an OpenAPI document as tools.

## Non-Goals

- XML responses, real image storage, or persistence across restarts.
- The legacy `api_key` and OAuth security schemes: the spec keeps them for
  reference, the server does not enforce them.

## Requirements

1. **Access levels.** Reads of the catalog need the agent's identity.
   Orders and user accounts need the person. Changing the catalog needs an
   auth token for `write:pets`, and placing or cancelling an order an auth
   token for `write:orders`, so the person approves each.
2. **Per-person data.** Orders belong to the person who placed them, keyed
   by the person's issuer and subject. A person cannot read another's order.
3. **Trust.** The server accepts person and auth tokens only from the
   person servers named with `--trust`.
4. **Spec-driven routes.** Every operation in the spec has a handler and an
   `x-aauth` annotation; the server refuses to start otherwise.
