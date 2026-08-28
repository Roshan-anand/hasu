package traefik

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Config files shipped with the binary; written to their output paths by
// SetupTraefik. Embedded so the single binary doesn't depend on the source
// tree at runtime.
//
//go:embed traefik.yml
var staticConfig []byte

//go:embed traefik.dynamic.yml
var dynamicConfig []byte

// SetupTraefik writes the embedded Traefik static and dynamic configs to
// their output paths, creating the parent directories if needed.
func SetupTraefik() error {
	staticPath := "/etc/hasu/traefik/traefik.yml"
	dynamicPath := "/etc/hasu/traefik/dynamic.yml"

	if err := os.MkdirAll(filepath.Dir(staticPath), 0755); err != nil {
		return fmt.Errorf("traefik: create static config dir: %w", err)
	}
	if err := os.WriteFile(staticPath, staticConfig, 0644); err != nil {
		return fmt.Errorf("traefik: write static config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dynamicPath), 0755); err != nil {
		return fmt.Errorf("traefik: create dynamic config dir: %w", err)
	}
	if err := os.WriteFile(dynamicPath, dynamicConfig, 0644); err != nil {
		return fmt.Errorf("traefik: write dynamic config: %w", err)
	}

	return nil
}
