// Package db handles new connections and transactions with timeouts.
//
// Timeouts that can be defined:
//    Connect timeout         - sets the maximum wait for creating a new connection
//    Statement timeout       - sets the maximum wait for queries completion
//    New transaction timeout - sets the maximum wait for opening a new transaction
//
// This feature is useful when the servers must be kept up, even though your database servers are
// not stable. So the database clients can respond their users with kindly error messages or look
// for other database servers - fail fast is usually better than keep server resources busy.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNewTxTimedOut returned when takes too long to retrieve a transaction from the database server.
var ErrNewTxTimedOut = errors.New("new transaction timed out")

// PostgresDriver holds the postgres driver name. You should import the driver somewhere in your
// code. The default value for PostgresDriver was get from https://github.com/lib/pq. This driver
// isn't directly imported here to don't stick this library with only one type of Postgres driver,
// and to make it more flexible for testing.
//
//     import (
//       _ "github.com/lib/pq"
//     )
var PostgresDriver = "postgres"

// ConnParams is all the data needed to create a new connection with database.
// If you are looking for default values see NewConnParams() function that set
// defaults values to Data structure.
type ConnParams struct {
	Username     string
	Password     string
	DatabaseName string
	Host         string

	// ConnectTimeout is the timeout utilized to creating a new connection with database. It is not
	// recommended to use a timeout of less than 2 seconds.
	ConnectTimeout time.Duration
	// StatementTimeout is the timeout utilized to invalidate queries that last more than the
	// configured timeout.
	StatementTimeout time.Duration

	MaxIdleConnections int
	MaxOpenConnections int
}

// NewConnParams returns the database connection parameters with some default
// values.
func NewConnParams() ConnParams {
	return ConnParams{
		Host:               "127.0.0.1",
		ConnectTimeout:     3 * time.Second,
		StatementTimeout:   10 * time.Second,
		MaxIdleConnections: 16,
		MaxOpenConnections: 32,
	}
}

// ConnectPostgres connects to a postgres database using the values from d. In case of a successfully
// connection it returns a sql.DB and a nil error. In case of problem it returns a nil sql.DB and an
// error from sql.Open (standard library, see https://golang.org/pkg/database/sql/#Open)
func ConnectPostgres(d ConnParams) (db *sql.DB, err error) {
	// connect_timeout
	//
	// https://www.postgresql.org/docs/9.6/static/libpq-connect.html#LIBPQ-CONNECT-CONNECT-TIMEOUT
	// Maximum wait for connection, in seconds (write as a decimal integer string). Zero or not
	// specified means wait indefinitely. It is not recommended to use a timeout of less than 2
	// seconds.
	//
	// statement_timeout
	//
	// https://www.postgresql.org/docs/9.6/static/runtime-config-client.html#GUC-STATEMENT-TIMEOUT
	// Abort any statement that takes more than the specified number of milliseconds, starting from
	// the time the command arrives at the server from the client.
	connParams := fmt.Sprintf(
		"user=%s password=%s dbname=%s sslmode=disable host=%s connect_timeout=%d statement_timeout=%d",
		d.Username,
		d.Password,
		d.DatabaseName,
		d.Host,
		int(d.ConnectTimeout.Seconds()),
		int(d.StatementTimeout.Seconds()*1000),
	)

	if db, err = sql.Open(PostgresDriver, connParams); err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(d.MaxIdleConnections)
	db.SetMaxOpenConns(d.MaxOpenConnections)
	return
}

// NewTx starts a new database transaction with a timeout. Starting a new transaction is not
// supposed to last too long, so a timeout of more than 3 seconds it's usually not necessary.
func NewTx(db *sql.DB, timeout time.Duration) (*sql.Tx, error) {
	if checker.checking() {
		return nil, ErrNewTxTimedOut // TODO change error type (?)
	}

	tx, err := newTx(db, timeout)
	if err == ErrNewTxTimedOut {
		go func() {
			checker.Check(db, timeout)
		}()
	}

	return tx, err
}

func newTx(db *sql.DB, timeout time.Duration) (*sql.Tx, error) {
	// The channels has size of 1 (buffered) to avoid keeping an unnecessary goroutine blocked in
	// memory. For example: a goroutine is spawn, and it returns via channel a new transaction or
	// an error. After spawning a goroutine the program blocks in the select statement waiting
	// until the first channel message. In case of a timeout message, the spawned goroutine will
	// put a message in one of this two channels (ch and chErr) and simply returns (die), the
	// program don't care about the messages, because it has already timed out. If the channels
	// were not buffered the goroutine would be blocked trying to put a message into the channel
	// until the program dies.
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

type dbChecker struct {
	sync.RWMutex
	checking bool
}

func (d *dbChecker) checking() bool {
	d.RLock()
	defer d.RUnlock()
	return d.checking
}

func (d *dbChecker) start() {
	d.Lock()
	defer d.Unlock()
	d.checking = true
}

func (d *dbChecker) stop() {
	d.Lock()
	defer d.Unlock()
	d.checking = false
}

func (d *dbChecker) Check(db *sql.DB, duration time.Duration) {
	if d.checking() {
		// already checking
		return
	}

	d.start()
	for range time.Tick(2 * time.Second) {
		tx, err := newTx(db, duration)
		if err != nil {
			continue
		}

		// db is back!
		d.stop()
		tx.Commit()
		return
	}
}

var checker = new(dbChecker)
