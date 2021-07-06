package sqlite3_js //nolint:golint

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
)

// SqliteJsConn implements driver.Conn.
type SqliteJsConn struct {
	JsDb js.Value // sql.js SQL.Database : https://sql-js.github.io/sql.js/documentation/class/Database.html
	mu   *sync.Mutex
}

// Prepare creates a prepared statement for later queries or executions. Multiple
// queries or executions may be run concurrently from the returned statement. The
// caller must call the statement's Close method when the statement is no longer
// needed.
func (conn *SqliteJsConn) Prepare(query string) (stmt driver.Stmt, err error) {
	defer protect("Prepare", func(e error) { err = e })
	return &SqliteJsStmt{
		c:  conn,
		js: conn.JsDb.Call("prepare", query),
	}, nil
}

// Close returns the connection to the connection pool. All operations after a
// Close will return with ErrConnDone. Close is safe to call concurrently with
// other operations and will block until all other operations finish. It may be
// useful to first cancel any used context and then call close directly after.
func (conn SqliteJsConn) Close() error {
	// TODO
	return nil
}

func (conn *SqliteJsConn) Exec(query string, args []driver.Value) (result driver.Result, err error) {
	defer protect("Exec", func(e error) { err = e })
	query = strings.TrimRight(query, ";")
	if strings.Contains(query, ";") {
		if len(args) != 0 {
			return nil, fmt.Errorf("cannot exec multiple statements with placeholders, query: %s nargs=%d", query, len(args))
		}
		jsVal, err := jsTryCatch(func() js.Value {
			return conn.JsDb.Call("exec", query)
		})
		if err != nil {
			return nil, err
		}
		return &SqliteJsResult{
			js:      jsVal,
			changes: 0,
			id:      0,
		}, nil
	}

	list := make([]namedValue, len(args))
	for i, v := range args {
		list[i] = namedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return conn.exec(context.Background(), query, list)
}

func (conn *SqliteJsConn) exec(ctx context.Context, query string, args []namedValue) (driver.Result, error) {
	// FIXME: we removed tbe ability to handle 'tails' - is this a problem?
	s, err := conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	var res driver.Result
	res, err = s.(*SqliteJsStmt).exec(ctx, args)
	s.Close()
	return res, err
}

// Transactions

// Begin starts a transaction. The default isolation level is dependent on the driver.
func (conn *SqliteJsConn) Begin() (driver.Tx, error) {
	return conn.begin(context.Background())
}

// BeginTx starts and returns a new transaction.
// If the context is canceled by the user the sql package will
// call Tx.Rollback before discarding and closing the connection.
//
// This must check opts.Isolation to determine if there is a set
// isolation level. If the driver does not support a non-default
// level and one is set or if there is a non-default isolation level
// that is not supported, an error must be returned.
//
// This must also check opts.ReadOnly to determine if the read-only
// value is true to either set the read-only transaction property if supported
// or return an error if it is not supported.
func (conn *SqliteJsConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return conn.begin(ctx)
}

func (conn *SqliteJsConn) begin(ctx context.Context) (driver.Tx, error) { //nolint:unparam
	/*
		if conn.disableTxns {
			fmt.Println("Ignoring BEGIN, txns disabled")
			return &SqliteJsTx{c: conn}, nil
		}
		if _, err := conn.exec(ctx, "BEGIN", nil); err != nil {
			return nil, err
		} */
	return &SqliteJsTx{c: conn}, nil
}
