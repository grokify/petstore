# ROADMAP — Petstore server with optional AAuth

**Initiative:** `INIT-PETSTORE-001`
**Repository:** `github.com/grokify/petstore`

Commits implementing an item carry `Refs: RMI-PETSTORE-<NNN>`.

## Phase 1 — Server

**Theme:** Every Petstore operation, served from the spec

- [ ] `RMI-PETSTORE-001` `x-aauth` annotations on every operation, with scope descriptions
- [ ] `RMI-PETSTORE-002` In-memory store and handlers for every operation, seeded with pets
- [ ] `RMI-PETSTORE-003` Spec-driven router that refuses an operation without a handler or annotation
  - Depends on: `RMI-PETSTORE-001`, `RMI-PETSTORE-002`
- [ ] `RMI-PETSTORE-004` AAuth mode with ResourceGuard, trusted person servers and per-person orders
  - Depends on: `RMI-PETSTORE-003`
- [ ] `RMI-PETSTORE-005` The spec served with servers pointing at the server itself
  - Depends on: `RMI-PETSTORE-003`

## Phase 2 — Run It

**Theme:** One command, one container

- [ ] `RMI-PETSTORE-006` `petstore-server` command
  - Depends on: `RMI-PETSTORE-004`
- [ ] `RMI-PETSTORE-007` Container image
  - Depends on: `RMI-PETSTORE-006`
- [ ] `RMI-PETSTORE-008` Guide: run it plain, with AAuth, and behind the aauth-go sandbox
  - Depends on: `RMI-PETSTORE-006`
