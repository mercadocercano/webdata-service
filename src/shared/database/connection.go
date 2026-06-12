package database

import (
	"database/sql"
	"time"

	"github.com/hornosg/go-shared/infrastructure/postgres"
)

const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
)

// NewPostgresDB opens and verifies a Postgres connection using the shared
// go-shared postgres helper, preserving webdata-service's pool tuning.
func NewPostgresDB(host, port, user, pass, dbName string) (*sql.DB, error) {
	db, err := postgres.Connect(postgres.Config{
		Host:            host,
		Port:            port,
		User:            user,
		Password:        pass,
		DBName:          dbName,
		SSLMode:         "disable",
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	})
	if err != nil {
		return nil, err
	}

	// postgres.Config does not expose ConnMaxIdleTime; preserve it here.
	db.SetConnMaxIdleTime(connMaxIdleTime)

	return db, nil
}
