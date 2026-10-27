// Package petstore holds the Petstore OpenAPI definition. Package server
// serves it, optionally as an AAuth resource.
package petstore

import _ "embed"

// OpenAPI is openapi.yaml: the Petstore API, with x-aauth annotations
// giving each operation's access level.
//
//go:embed openapi.yaml
var OpenAPI []byte
