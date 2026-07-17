package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Roshan-anand/godploy/internal/db"
	localSql "github.com/Roshan-anand/godploy/sqlite"

	"github.com/golang-migrate/migrate/v4"
	migrateSqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/mattn/go-sqlite3"
)

type DataBase struct {
	Pool    *sql.DB
	Queries *db.Queries
}

const (
	MAX_DB_OPEN_CONNECTIONS = 1
	MAX_DB_IDLE_CONNECTIONS = 1
	PING_TIMEOUT            = 5
)

var Pool_Close_Err = fmt.Errorf("DB pool close err")

// for migrating the database
func MigrateDb(db *sql.DB) error {
	mFs, err := localSql.GetMigrationFS()
	if err != nil {
		return err
	}

	source, err := iofs.New(mFs, ".")
	if err != nil {
		return err
	}

	driver, err := migrateSqlite.WithInstance(db, &migrateSqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		source,
		"sqlite3",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	fmt.Println("database migrated completed ...")
	return nil
}

// initialize and return a new database connection
func InitDb(dir string) (*DataBase, error) {
	// if directory doesn't exist create it
	if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	p := filepath.Join(dir, "base.db")
	dsn := "file:" + p +
		"?_pragma=journal_mode(WAL)" +
		"&_foreign_keys=1" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)"
	pool, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	pool.SetMaxOpenConns(MAX_DB_OPEN_CONNECTIONS)
	pool.SetMaxIdleConns(MAX_DB_IDLE_CONNECTIONS)

	// run migrations
	if err := MigrateDb(pool); err != nil {
		if cErr := pool.Close(); cErr != nil {
			return nil, errors.Join(err, cErr)
		}
		return nil, fmt.Errorf("Migration error : %w", err)
	}

	// ping the database to ensure connection is established
	ctx, cancle := context.WithTimeout(context.Background(), PING_TIMEOUT*time.Second)
	defer cancle()

	if err := pool.PingContext(ctx); err != nil {
		if cErr := pool.Close(); cErr != nil {
			return nil, errors.Join(Pool_Close_Err, err, cErr)
		}
		return nil, errors.Join(Pool_Close_Err, err)
	}

	queries := db.New(pool) // get query instance from sqlc generated code

	fmt.Println("database connection established ...") // TODO : replace with proper logging
	return &DataBase{
		Pool:    pool,
		Queries: queries,
	}, nil
}

// close the database connection
func (db *DataBase) CloseDb() error {
	fmt.Println("closing database connection")
	if err := db.Pool.Close(); err != nil {
		return errors.Join(Pool_Close_Err, err)
	}

	return nil
}

// helper function to check if UNIQUE constraint error
func (_ *DataBase) IsUniqueConstraintError(err error) bool {
	var sqlErr sqlite3.Error
	return errors.As(err, &sqlErr) && (sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique)
}

// helper function to check if UNIQUE constraint error
func (_ *DataBase) IsNoRowsError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
