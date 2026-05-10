package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Roshan-anand/godploy/internal/lib/sse"
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

func (db *BadgerDB) AddLogs(dID uuid.UUID, logs []string) {
	txn := db.Pool.NewTransaction(true)

	for i, log := range logs {
		key := fmt.Sprintf("%s_%d", dID.String(), i)
		if err := txn.Set([]byte(key), []byte(log)); err == badger.ErrTxnTooBig {
			_ = txn.Commit()
			txn = db.Pool.NewTransaction(true)
			_ = txn.Set([]byte(key), []byte(log))
		}
	}
	_ = txn.Commit()
}

// get all logs of a deployment by deployment id
func (db *BadgerDB) StreamAllLogsByDeploymentID(dID uuid.UUID, sse *sse.SSE) error {
	prefix := []byte(dID.String() + "_")

	err := db.Pool.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = prefix

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			// k := item.Key()
			if err := item.Value(func(val []byte) error {
				sse.SendEvent("log", val)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// delete all logs of a deployment by deployment id
func (db *BadgerDB) DeleteAllLogsByDeploymentID(dIDs []uuid.UUID) error {
	var err error
	for _, id := range dIDs {

		prefix := []byte(id.String() + "_")

		if updErr := db.Pool.Update(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			opts.Prefix = prefix

			it := txn.NewIterator(opts)
			defer it.Close()

			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				key := it.Item().KeyCopy(nil)
				if delErr := txn.Delete(key); delErr != nil {
					return err
				}
			}

			return nil
		}); updErr != nil {
			err = updErr
			continue
		}

	}
	return err
}
