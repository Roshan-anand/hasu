package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// returns the embedded filesystem for the frontend dist directory
func GetEmbedFS() (fs.FS, error) {
	DistDirFS, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, err
	}

	return DistDirFS, nil
}
