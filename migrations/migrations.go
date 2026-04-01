package migrations

import "embed"

//go:embed postgres/*.sql
var PostgresFS embed.FS
