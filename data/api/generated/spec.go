// Package generated embeds the generated API contract in the RPC binary.
// Regenerate Swagger before building to keep API resource synchronization current.
package generated

import _ "embed"

// Swagger is the trusted build-time API contract; clients cannot supply one.
//
//go:embed go-zero-admin.swagger.json
var Swagger []byte
