// Package api embeds the OpenAPI spec so the server can publish it.
package api

import _ "embed"

// Spec is the OpenAPI 3 document (api/openapi.yaml).
//
//go:embed openapi.yaml
var Spec []byte
