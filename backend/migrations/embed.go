package migrations

import "embed"

// Files embeds SQL migrations so the backend can run as a single binary later.
//
//go:embed *.sql
var Files embed.FS
