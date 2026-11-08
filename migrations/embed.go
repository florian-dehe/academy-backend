// Package migrations embeds the versioned SQL migrations (goose format).
// They are also the schema source for sqlc (see sqlc.yaml).
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
