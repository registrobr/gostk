// Package db handles new connections and new transactions with database
package db

import (
	"time"
	"errors"
	"fmt"
	"database/sql"
)

var ErrNewTxTimedOut = errors.New("new transaction timed out")

// Data is all the data needed to create a new connection with
type Data struct {
	Username string
	Password string
	DatabaseName string
	Host string

	ConnectTimeout time.Duration
	StatementTimeout time.Duration

	MaxIdleConnections int
	MaxOpenConnections int
}

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

	if db, err = sql.Open("postgres", connParams); err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(16)
	db.SetMaxOpenConns(32)

	return db, err
}

func newTx(db *sql.DB, ch chan *sql.Tx, chErr chan error) {
	tx, err := db.Begin()
	if err != nil {
		chErr <- err
		return
	}

	ch <- tx
}

func NewTx(db *sql.DB, timeout time.Duration) (tx *sql.Tx, err error)  {
	ch := make(chan *sql.Tx, 1)
	chErr := make(chan error, 1)
	go newTx(db, ch, chErr)

	select {
	case tx = <-ch:
		return tx, nil
	case <-time.After(timeout):
		return nil, ErrNewTxTimedOut
	}

	return nil, ErrNewTxTimedOut
}