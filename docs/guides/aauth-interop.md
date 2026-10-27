# Test AAuth Interop

Use the Petstore as an AAuth resource to check that an agent, a person server
or an agent provider of your own works with the others. The servers around it
come from the AAuth sandbox in [aauth-go](https://github.com/aauth-dev/aauth-go),
which has the full
[sandbox guide](https://aauth-dev.github.io/aauth-go/guides/sandbox/); this page
covers what is specific to the Petstore.

## What Each Operation Asks For

In AAuth mode (`--auth aauth`) each operation requires what its `x-aauth`
annotation in `openapi.yaml` says, so one resource exercises every level:

| Level | Operations | An agent must |
|-------|------------|---------------|
| Agent identity | `findPetsByStatus`, `findPetsByTags`, `getPetById`, `getInventory`, `loginUser`, `logoutUser` | sign the request with its agent token |
| Person | `getOrderById`, the user operations | present a person token |
| Auth token, `write:orders` | `placeOrder`, `deleteOrder` | get the person's approval, then present the auth token |
| Auth token, `write:pets` | `addPet`, `updatePet`, `updatePetWithForm`, `deletePet`, `uploadFile` | the same, for `write:pets` |

Orders belong to the person who placed them: another person's order is not
found. The resource metadata, with the scope descriptions a person server shows
on its consent page, is at `/.well-known/aauth-resource.json`.

## Run It with the Sandbox

The [aauth-postman](https://github.com/aistandardsio/aauth-postman) repository has
a Compose file that runs the sandbox and the Petstore together, with the
Petstore fronted at `https://petstore.aauth.test`:

```bash
cd aauth-postman
docker compose up -d --build --wait
docker compose --profile test run --rm newman-petstore   # a reference agent
```

The Compose file builds the Petstore image from this repository, found at
`../../grokify/petstore` unless you set `PETSTORE_DIR`. To run the pieces by hand, see the flags in the
[README](https://github.com/grokify/petstore#run-it-as-an-aauth-resource).

## Check Your Agent

Enroll at the sandbox's agent provider, then call the Petstore. A correct agent
sees these responses, in order:

1. `GET /v2/pet/findByStatus?status=available` signed with the agent token:
   **200**.
2. `POST /v2/store/order`: **401**, `AAuth-Requirement: requirement=person-token`.
3. The same request with a person token: **401**,
   `requirement=auth-token`, with a resource token for `write:orders`.
4. The resource token sent to the person server's token endpoint: **202** with
   an interaction code while the person approves, then **200** with the auth
   token.
5. The order repeated with the auth token: **200**.

The sandbox records every message:

```bash
docker compose exec sandbox aauth-sandbox trace --state /state
```

Two reference agents send these messages. The Postman collection shows each as
one request, and [omniagent](https://plexusone.github.io/omniagent/guides/aauth/try-it/)
is an AI agent that does the whole exchange itself. Comparing your agent's
trace with theirs shows where it differs.

## Check Your Own Server Instead

To test a person server or agent provider of your own, start the sandbox with
`--remote` ([Test another implementation](https://aauth-dev.github.io/aauth-go/guides/sandbox/#test-another-implementation))
and run the same agent against the Petstore. The Petstore trusts the person
server named by `--trust`, so give it your server's origin.

## Reset the Store

The store is in memory and starts with six pets. A pet that an order sells is
gone until the server restarts:

```bash
docker compose up -d --force-recreate petstore
```
