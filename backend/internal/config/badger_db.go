package config

import (
	"fmt"
	"os"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
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

func (db *BadgerDB) GetAllLogsByDeploymentID(dID uuid.UUID) ([]string, error) {
	prefix := []byte(dID.String() + "_")

	logs := []string{}

	err := db.Pool.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		// Optional: narrows the iterator to this prefix
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			fmt.Printf("%s -> %s\n", key, val)
			logs = append(logs, string(val))
		}

		return nil
	})

	return logs, err
}
