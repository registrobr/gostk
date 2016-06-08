// Package db handles new connections and new transactions with database adding an extra layer to
// deal with timeouts and other connection problems during the operation.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNewTxTimedOut returned when takes too long to retrieve a transaction from the database server.
var ErrNewTxTimedOut = errors.New("new transaction timed out")

// PostgresDriver used to connect using a postgres driver. You should import it somewhere in your
// code. By default "postgres" is used from github.com/lib/pq. We don't import directly here to
// don't stick this library with only one type of Postgres driver, and to make it more flexible for
// testing.
//
//     import (
//       _ "github.com/lib/pq"
//     )
var PostgresDriver = "postgres"

// Data is all the data needed to create a new connection
type Data struct {
	Username     string
	Password     string
	DatabaseName string
	Host         string

	ConnectTimeout   time.Duration
	StatementTimeout time.Duration

	MaxIdleConnections int
	MaxOpenConnections int
}

// NewData returns the database connection parameters with some default values.
func NewData() Data {
	return Data{
		Host:               "127.0.0.1",
		ConnectTimeout:     3 * time.Second,
		StatementTimeout:   10 * time.Second,
		MaxIdleConnections: 16,
		MaxOpenConnections: 32,
	}
}

// ConnectPostgres used to connect to any Postgres database
func ConnectPostgres(d Data) (db *sql.DB, err error) {
	connParams := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=disable host=%s connect_timeout=%d statement_timeout=%d",
		d.Username,
		d.Password,
		d.DatabaseName,
		d.Host,
		int(d.ConnectTimeout.Seconds()),
		int(d.StatementTimeout.Seconds()),
	)

	if db, err = sql.Open(PostgresDriver, connParams); err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(d.MaxIdleConnections)
	db.SetMaxOpenConns(d.MaxOpenConnections)
	return
}

// NewTx starts a new database transaction adding an extra layer to deal with database timeouts
// during the action.
func NewTx(db *sql.DB, timeout time.Duration) (*sql.Tx, error) {
	ch := make(chan *sql.Tx, 1)
	chErr := make(chan error, 1)

	go func() {
		tx, err := db.Begin()
		if err != nil {
			chErr <- err
			return
		}

		ch <- tx
	}()

	select {
	case tx := <-ch:
		return tx, nil
	case err := <-chErr:
		return nil, err
	case <-time.After(timeout):
		return nil, ErrNewTxTimedOut
	}
}
