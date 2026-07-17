package migration

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var embeded embed.FS

// returns the embedded filesystem for the migration files
func GetMigrationFS() (fs.FS, error) {
	MigrationFS, err := fs.Sub(embeded, "migrations")
	if err != nil {
		return nil, err
	}
	return MigrationFS, nil
}
