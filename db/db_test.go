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
				connParams := "user=user password=passwd dbname=test sslmode=disable host=127.0.0.1 port=5432 connect_timeout=3 statement_timeout=10000"
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
		{
			description: "check if it is setting default postgres port 5432",
			connParams: func() db.ConnParams {
				d := db.NewConnParams()
				d.DatabaseName = "test"
				d.Username = "user"
				d.Password = "passwd"
				d.Host = "127.0.0.1"
				d.Port = 0
				return d
			}(),
			postgresDriver: "testdb",
			openFunc: func(dsn string) (driver.Conn, error) {
				connParams := "user=user password=passwd dbname=test sslmode=disable host=127.0.0.1 port=5432 connect_timeout=3 statement_timeout=10000"
				if dsn != connParams {
					return nil, fmt.Errorf("invalid connection string")
				}

				return testdb.Conn(), nil
			},
		},
	}

	originalPostgresDriver := db.PostgresDriver
	defer func() {
		db.PostgresDriver = originalPostgresDriver
	}()

	defer func() {
		testdb.SetOpenFunc(nil)
	}()

	for i, scenario := range scenarios {
		testdb.SetOpenFunc(scenario.openFunc)
		db.PostgresDriver = scenario.postgresDriver
		db, err := db.ConnectPostgres(scenario.connParams)

		// call Begin to force database/sql call our scenario.openFunc
		if err == nil {
			_, err = db.Begin()
		}

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

func TestDB_Begin(t *testing.T) {
	scenarios := []struct {
		description   string
		db            *db.DB
		beginFunc     func() (driver.Tx, error)
		rerun         int
		rerunDelay    time.Duration
		afterDelay    time.Duration
		expectedError error
	}{
		{
			description: "it should initialize a transaction correctly",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 1*time.Second)
			}(),
			beginFunc: func() (driver.Tx, error) {
				return &testdb.Tx{}, nil
			},
		},
		{
			description: "it should detect an error while initializing a transaction",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 1*time.Second)
			}(),
			beginFunc: func() (driver.Tx, error) {
				return nil, fmt.Errorf("i'm a crazy error")
			},
			expectedError: fmt.Errorf("i'm a crazy error"),
		},
		{
			description: "it should timeout when transaction takes too long to start",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 10*time.Millisecond)
			}(),
			beginFunc: func() (driver.Tx, error) {
				time.Sleep(1 * time.Second)
				return &testdb.Tx{}, nil
			},
			expectedError: db.ErrNewTxTimedOut,
		},
		{
			description: "it should fail fast when the database is unreachable",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 10*time.Millisecond)
			}(),
			beginFunc: func() func() (driver.Tx, error) {
				i := 0
				return func() (driver.Tx, error) {
					i++
					if i <= 2 {
						// force timeout
						time.Sleep(1 * time.Second)
					}
					return &testdb.Tx{}, nil
				}
			}(),
			rerun:         1,
			rerunDelay:    10 * time.Millisecond,
			expectedError: db.ErrUnreachable,
		},
		{
			description: "it should restore from a database unreachable problem",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 10*time.Millisecond)
			}(),
			beginFunc: func() func() (driver.Tx, error) {
				i := 0
				return func() (driver.Tx, error) {
					i++
					if i <= 2 {
						// force timeout
						time.Sleep(1 * time.Second)
					}
					return &testdb.Tx{}, nil
				}
			}(),
			rerun:      41,
			rerunDelay: 100 * time.Millisecond,
		},
		{
			// this test is only made to try getting some panic in a concurrent
			// scenario and to achieve 100% test coverage. =)
			description: "it should avoid running 2 checkers at once",
			db: func() *db.DB {
				d, err := sql.Open("testdb", "")
				if err != nil {
					t.Fatal(err)
				}
				return db.NewDB(d, 10*time.Millisecond)
			}(),
			beginFunc: func() func() (driver.Tx, error) {
				i := 0
				return func() (driver.Tx, error) {
					i++
					if i <= 2 {
						// force timeout
						time.Sleep(1 * time.Second)
					}
					return &testdb.Tx{}, nil
				}
			}(),
			rerun:         2,
			afterDelay:    20 * time.Millisecond,
			expectedError: db.ErrUnreachable,
		},
	}

	defer func() {
		testdb.SetBeginFunc(nil)
	}()

	for i, scenario := range scenarios {
		testdb.SetBeginFunc(scenario.beginFunc)

		tx, err := scenario.db.Begin()
		for j := 0; j < scenario.rerun; j++ {
			time.Sleep(scenario.rerunDelay)
			tx, err = scenario.db.Begin()
		}

		// in some scenarios we want to wait before finishing the test so the go
		// routines can run at least one time.
		time.Sleep(scenario.afterDelay)

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

func TestUnreachable(t *testing.T) {
	rawDB, err := sql.Open("testdb", "")
	if err != nil {
		t.Fatal(err)
	}
	d := db.NewDB(rawDB, 10*time.Millisecond)

	defer func() {
		testdb.SetBeginFunc(nil)
	}()

	i := 0
	testdb.SetBeginFunc(func() (driver.Tx, error) {
		i++
		if i <= 1 {
			time.Sleep(1 * time.Second)
		}
		return &testdb.Tx{}, nil
	})

	if _, err := d.Begin(); err == nil {
		t.Error("timeout not detected")
	}

	// wait a bit for the dbChecker to start
	time.Sleep(10 * time.Millisecond)
	if !db.Unreachable(d) {
		t.Error("should be unreachable")
	}

	// wait for the dbChecker to run again
	time.Sleep(4 * time.Second)
	if db.Unreachable(d) {
		t.Error("should be reachable")
	}

	if db.Unreachable(nil) {
		t.Error("not detecting nil DB")
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

	conn := db.NewDB(dbConn, 3*time.Second)
	fmt.Println(conn != nil)

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

	tx, err := db.NewDB(dbConn, 3*time.Second).Begin()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(tx != nil)

	// Output:
	// true
}
