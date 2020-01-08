// -*- coding: utf-8 -*-
// Copyright 2020 The Matrix.org Foundation C.I.C.
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import "syscall/js"

func init() {
	sql.Register("sqlite-js", &SqliteJsDriver{})
}

// SqliteJsDriver implements driver.Driver.
type SqliteJsDriver struct {
	ConnectHook func(*SqliteJsConn) error
}

// SqliteJsConn implements driver.Conn.
type SqliteJsConn struct {
	js.Value JsDb

}

// SqliteJsTx implements driver.Tx.
type SqliteJsTx struct {
	c *SqliteJsConn
}

// SqliteJsStmt implements driver.Stmt.
type SqliteJsStmt struct {
	c      *SqliteJsConn
	t      string
	closed bool
	cls    bool
}

// SqliteJsResult implements sql.Result.
type SqliteJsResult struct {
	id      int64
	changes int64
}

// SqliteJsRows implements driver.Rows.
type SqliteJsRows struct {
	s        *SqliteJsStmt
	nc       int
	cols     []string
	decltype []string
	cls      bool
	closed   bool
	ctx      context.Context // no better alternative to pass context into Next() method
}

// Database conns
func (d *SQLiteDriver) Open(dsn string) (driver.Conn, error) {
	bridge := js.Global().Get("bridge")
	jsDb = bridge.Call("open", dsn)
	return SqliteJsConn{jsDb}, nil
}

/*
Exec executes a query without returning any rows. The args are for any
placeholder parameters in the query.
*/
func (conn *SqliteJsConn) Exec(query string, args ...interface{}) (driver.Result, error) {
	bridge := js.Global().Get("bridge")
	bridge.Call("exec", conn.JsDb, query, args)
	return driver.Result{}, nil
}


// Transactions

/*
Begin starts a transaction. The default isolation level is dependent on the driver.
*/
func (conn *SqliteJsConn) Begin() (*driver.Tx, error)

/*
BeginTx starts a transaction.

The provided context is used until the transaction is committed or rolled
back. If the context is canceled, the sql package will roll back the
transaction. Tx.Commit will return an error if the context provided to BeginTx
is canceled.

The provided TxOptions is optional and may be nil if defaults should be used.
If a non-default isolation level is used that the driver doesn't support, an
error will be returned.
*/
func (conn *SqliteJsConn) BeginTx(ctx context.Context, opts *driver.TxOptions) (*driver.Tx, error)

/*
Commit commits the transaction.
*/
func (tx *SqliteJsTx) Commit() error

/*
Rollback aborts the transaction.
*/
func (tx *SqliteJsTx) Rollback() error

/*
Stmt returns a transaction-specific prepared statement from an existing statement.
*/
func (tx *SqliteJsTx) Stmt(stmt *driver.Stmt) *driver.Stmt


// Statements

/*
Prepare creates a prepared statement for later queries or executions. Multiple
queries or executions may be run concurrently from the returned statement. The
caller must call the statement's Close method when the statement is no longer
needed.
*/
func (conn *SqliteJsConn) Prepare(query string) (*driver.Stmt, error)

/*
ExecContext executes a prepared statement with the given arguments and returns
a Result summarizing the effect of the statement. 
*/
func (s *SqliteJsStmt) ExecContext(ctx context.Context, args ...interface{}) (driver.Result, error)

/*
QueryContext executes a prepared query statement with the given arguments and
returns the query results as a *Rows. 
*/
func (s *SqliteJsStmt) QueryContext(ctx context.Context, args ...interface{}) (*driver.Rows, error)

/*
QueryRowContext executes a prepared query statement with the given arguments.
If an error occurs during the execution of the statement, that error will be
returned by a call to Scan on the returned *Row, which is always non-nil. If
the query selects no rows, the *Row's Scan will return ErrNoRows. Otherwise,
the *Row's Scan scans the first selected row and discards the rest.
*/
func (s *SqliteJsStmt) QueryRowContext(ctx context.Context, args ...interface{}) *driver.Row


// Rows

/*
Scan copies the columns in the current row into the values pointed at by dest.
The number of values in dest must be the same as the number of columns in
Rows.
*/
func (rs *SqliteJsRows) Scan(dest ...interface{}) error

/*
Next prepares the next result row for reading with the Scan method. It returns
true on success, or false if there is no next result row or an error happened
while preparing it. Err should be consulted to distinguish between the two
cases.

Every call to Scan, even the first one, must be preceded by a call to Next.
*/
func (rs *SqliteJsRows) Next() bool

/*
Close closes the Rows, preventing further enumeration. If Next is called and
returns false and there are no further result sets, the Rows are closed
automatically and it will suffice to check the result of Err. Close is
idempotent and does not affect the result of Err. 
*/
func (rs *SqliteJsRows) Close() error


// Results

// LastInsertId return last inserted ID.
func (r *SqliteJsResult) LastInsertId() (int64, error) {
	return r.id, nil
}

// RowsAffected return how many rows affected.
func (r *SqliteJsResult) RowsAffected() (int64, error) {
	return r.changes, nil
}
