package migrations

import "embed"

//go:embed "*.sql"
var MigrationsFiles embed.FS
