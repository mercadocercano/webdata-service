package webdata

import "embed"

// MigrationsFS embeds all migration files for webdata-service.
// The "migrations" subdirectory name is required by the go-shared migrate helper
// (iofs.New expects the files under a named subdirectory of the provided FS).
//
// This file lives at the module root so that //go:embed can reference the
// sibling migrations/ directory — cmd/api/main.go cannot embed it directly
// because //go:embed does not support paths that escape the package directory.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
