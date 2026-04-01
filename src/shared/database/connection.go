package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
)

func NewPostgresDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening postgres connection: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	return db, nil
}

func BuildConnString(host, port, user, pass, dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbName,
	)
}
