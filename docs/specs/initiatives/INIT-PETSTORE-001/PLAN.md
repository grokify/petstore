# PLAN — Petstore server with optional AAuth

**Initiative:** `INIT-PETSTORE-001`
**Repository:** `github.com/grokify/petstore`

1. Annotate the spec with `x-aauth` and scope descriptions.
2. Build the store and the handlers for all operations, then the
   spec-driven router, with tests that every operation is routed.
3. Add AAuth mode with aauth-go's `ResourceGuard`, tested end to end with
   aauth-go's `Transport` against an in-process person server.
4. Add the `petstore-server` command, the image, and the guide.

The server depends on aauth-go v0.3.0.
