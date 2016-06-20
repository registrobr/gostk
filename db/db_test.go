package db_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"

	testdb "github.com/erikstmartin/go-testdb"
	"github.com/registrobr/gostk/db"
)

func TestConnectPostgres(t *testing.T) {
	scenarios := []struct {
		description    string
		connParams     db.ConnParams
		postgresDriver string
		openFunc       func(dsn string) (driver.Conn, error)
		expectedError  error
	}{
		{
			description: "it should connect correctly to the database",
			connParams: func() db.ConnParams {
				d := db.NewConnParams()
				d.DatabaseName = "test"
				d.Username = "user"
				d.Password = "passwd"
				return d
			}(),
			postgresDriver: "testdb",
			openFunc: func(dsn string) (driver.Conn, error) {
				connParams := "user=user password=passwd dbname=test sslmode=disable host=127.0.0.1 connect_timeout=3 statement_timeout=10000"
				if dsn != connParams {
					return nil, fmt.Errorf("invalid connection string")
				}

				return testdb.Conn(), nil
			},
		},
		{
			description: "it should detect an unknown driver",
			connParams: func() db.ConnParams {
				d := db.NewConnParams()
				d.DatabaseName = "test"
				d.Username = "user"
				d.Password = "passwd"
				return d
			}(),
			postgresDriver: "idontexist",
			expectedError:  fmt.Errorf(`sql: unknown driver "idontexist" (forgotten import?)`),
		},
	}

	originalPostgresDriver := db.PostgresDriver
	defer func() {
		db.PostgresDriver = originalPostgresDriver
	}()

	for i, scenario := range scenarios {
		testdb.SetOpenFunc(scenario.openFunc)
		db.PostgresDriver = scenario.postgresDriver
		db, err := db.ConnectPostgres(scenario.connParams)

		if scenario.expectedError == nil && db == nil {
			t.Errorf("scenario %d, “%s”: database not initialized",
				i, scenario.description)
		}

		if !reflect.DeepEqual(err, scenario.expectedError) {
			t.Errorf("scenario %d, “%s”: mismatch errors. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expectedError, err)
		}
	}
}

func TestNewTx(t *testing.T) {
	scenarios := []struct {
		description   string
		db            *sql.DB
		beginFunc     func() (driver.Tx, error)
		timeout       time.Duration
		expectedError error
	}{
		{
			description: "it should initialize a transaction correctly",
			db: func() *sql.DB {
				db, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			beginFunc: func() (driver.Tx, error) {
				return &testdb.Tx{}, nil
			},
			timeout: 1 * time.Second,
		},
		{
			description: "it should detect an error while initializing a transaction",
			db: func() *sql.DB {
				db, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			beginFunc: func() (driver.Tx, error) {
				return nil, fmt.Errorf("i'm a crazy error")
			},
			timeout:       1 * time.Second,
			expectedError: fmt.Errorf("i'm a crazy error"),
		},
		{
			description: "it should timeout when transaction takes too long to start",
			db: func() *sql.DB {
				db, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db
			}(),
			beginFunc: func() (driver.Tx, error) {
				time.Sleep(1 * time.Second)
				return &testdb.Tx{}, nil
			},
			timeout:       10 * time.Millisecond,
			expectedError: db.ErrNewTxTimedOut,
		},
	}

	for i, scenario := range scenarios {
		testdb.SetBeginFunc(scenario.beginFunc)
		tx, err := db.NewTx(scenario.db, scenario.timeout)

		if scenario.expectedError == nil && tx == nil {
			t.Errorf("scenario %d, “%s”: tx not initialized",
				i, scenario.description)
		}

		if !reflect.DeepEqual(err, scenario.expectedError) {
			t.Errorf("scenario %d, “%s”: mismatch errors. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expectedError, err)
		}
	}
}

func ExampleConnectPostgres() {
	db.PostgresDriver = "testdb"

	params := db.ConnParams{
		Username:           "user",
		Password:           "passwd",
		DatabaseName:       "dbname",
		Host:               "localhost:5432",
		ConnectTimeout:     3 * time.Second,
		StatementTimeout:   10 * time.Second,
		MaxIdleConnections: 16,
		MaxOpenConnections: 32,
	}

	dbConn, err := db.ConnectPostgres(params)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(dbConn != nil)

	// Output:
	// true
}

func ExampleNewTx() {
	db.PostgresDriver = "testdb"

	params := db.ConnParams{
		Username:           "user",
		Password:           "passwd",
		DatabaseName:       "dbname",
		Host:               "localhost:5432",
		ConnectTimeout:     3 * time.Second,
		StatementTimeout:   10 * time.Second,
		MaxIdleConnections: 16,
		MaxOpenConnections: 32,
	}

	// get dbConn from a global variable or a local pool
	dbConn, err := db.ConnectPostgres(params)
	if err != nil {
		fmt.Println(err)
		return
	}

	tx, err := db.NewTx(dbConn, 3*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(tx != nil)

	// Output:
	// true
}
