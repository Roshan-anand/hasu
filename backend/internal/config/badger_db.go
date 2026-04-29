package config

import (
	"fmt"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
)

type BadgerDB struct {
	Pool *badger.DB
}

func InitBadgerDB(dir string) (*BadgerDB, error) {
	// if directory doesn't exist create it
	if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	p := filepath.Join(dir, "logs")

	pool, err := badger.Open(badger.DefaultOptions(p))
	if err != nil {
		return nil, fmt.Errorf("failed to open badgerDB: %w", err)
	}

	fmt.Println("badgerDB connection established ...")
	return &BadgerDB{Pool: pool}, nil
}

// close the database connection
func (db *BadgerDB) CloseDb() error {
	fmt.Println("closing database connection")
	return db.Pool.Close()
}
