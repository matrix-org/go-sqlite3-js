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

// Inspired by https://github.com/mattn/go-sqlite3

package main

import "database/sql"
import "database/sql/driver"
import "syscall/js"
import "context"
import "fmt"

func init() {
	sql.Register("sqlite-js", &SqliteJsDriver{})
}

// SqliteJsDriver implements driver.Driver.
type SqliteJsDriver struct {
	ConnectHook func(*SqliteJsConn) error
}

// SqliteJsConn implements driver.Conn.
type SqliteJsConn struct {
	JsDb js.Value
}

// SqliteJsTx implements driver.Tx.
type SqliteJsTx struct {
	c *SqliteJsConn
}

// SqliteJsStmt implements driver.Stmt.
type SqliteJsStmt struct {
	c      *SqliteJsConn
	js     js.Value
	t      string
	closed bool
	cls    bool
}

// SqliteJsResult implements sql.Result.
type SqliteJsResult struct {
	js      js.Value
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
func (d *SqliteJsDriver) Open(dsn string) (driver.Conn, error) {
	bridge := js.Global().Get("bridge")
	jsDb := bridge.Call("open", dsn)
	return &SqliteJsConn{jsDb}, nil
}


// Transactions


// Begin starts a transaction. The default isolation level is dependent on the driver.
func (conn *SqliteJsConn) Begin() (driver.Tx, error) {
	return nil, nil
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
	return nil, nil
}

// Close returns the connection to the connection pool. All operations after a
// Close will return with ErrConnDone. Close is safe to call concurrently with
// other operations and will block until all other operations finish. It may be
// useful to first cancel any used context and then call close directly after.
func (conn SqliteJsConn) Close() error {
	return nil
}

// Commit commits the transaction.
func (tx *SqliteJsTx) Commit() error {
	return nil
}

// Rollback aborts the transaction.
func (tx *SqliteJsTx) Rollback() error {
	return nil
}


// Statements


// Prepare creates a prepared statement for later queries or executions. Multiple
// queries or executions may be run concurrently from the returned statement. The
// caller must call the statement's Close method when the statement is no longer
// needed.
func (conn *SqliteJsConn) Prepare(query string) (driver.Stmt, error) {
	bridge := js.Global().Get("bridge")
	jsStmt := bridge.Call("prepare", conn.JsDb, query)
	return &SqliteJsStmt{
		c: conn,
		js: jsStmt,
	}, nil
}

type namedValue struct {
	Name    string
	Ordinal int
	Value   driver.Value
}

// Exec executes a prepared statement with the given arguments and returns a
// Result summarizing the effect of the statement. 
func (s *SqliteJsStmt) Exec(args []driver.Value) (driver.Result, error) {
	list := make([]namedValue, len(args))
	for i, v := range args {
		list[i] = namedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return s.exec(context.Background(), list)
}

// ExecContext executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
func (s *SqliteJsStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	list := make([]namedValue, len(args))
	for i, nv := range args {
		list[i] = namedValue(nv)
	}
	return s.exec(ctx, list)
}

func (s *SqliteJsStmt) exec(ctx context.Context, args []namedValue) (driver.Result, error) {
	bridge := js.Global().Get("bridge")
	jsArgs := make([]interface{}, len(args) + 1)
	jsArgs[0] = s.js
	for i, v := range args {
		jsArgs[i + 1] = js.ValueOf(v.Value)
	}
	res := bridge.Call("exec", jsArgs...)
	return &SqliteJsResult{js: res}, nil
}


// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or additionally).
func (s *SqliteJsStmt) Query(args []driver.Value) (driver.Rows, error) {
	list := make([]namedValue, len(args))
	for i, v := range args {
		list[i] = namedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return s.query(context.Background(), list)
}

// QueryContext executes a query that may return rows, such as a
// SELECT.
//
// QueryContext must honor the context timeout and return when it is canceled.
func (s *SqliteJsStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	list := make([]namedValue, len(args))
	for i, nv := range args {
		list[i] = namedValue(nv)
	}
	return s.query(ctx, list)
}

func (s *SqliteJsStmt) query(ctx context.Context, args []namedValue) (driver.Rows, error) {
	bridge := js.Global().Get("bridge")
	jsArgs := make([]interface{}, len(args) + 1)
	jsArgs[0] = s.js
	for i, v := range args {
		jsArgs[i + 1] = js.ValueOf(v.Value)
	}
	res := bridge.Call("query", jsArgs...)
	if res.Bool() == false {
		return nil, fmt.Errorf("couldn't bind params to query")
	}

	return &SqliteJsRows{s: s}, nil
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (s *SqliteJsStmt) NumInput() int {
	return -1
}

// Close closes the statement.
func (s *SqliteJsStmt) Close() error {
	bridge := js.Global().Get("bridge")
	res := bridge.Call("close", s.js)
	if res.Bool() == false {
		return fmt.Errorf("couldn't close stmt")
	}
	return nil
}


// Rows


// Columns returns the names of the columns. The number of
// columns of the result is inferred from the length of the
// slice. If a particular column name isn't known, an empty
// string should be returned for that entry.
func (r *SqliteJsRows) Columns() []string {
	bridge := js.Global().Get("bridge")
	res := bridge.Call("columns", r.s.js)
	cols := make([]string, res.Length())
	for i := 0; i <= res.Length(); i++ {
		cols[i] = res.Get(string(i)).String()
	}
	return cols
}

// Next is called to populate the next row of data into
// the provided slice. The provided slice will be the same
// size as the Columns() are wide.
//
// Next should return io.EOF when there are no more rows.
//
// The dest should not be written to outside of Next. Care
// should be taken when closing Rows not to modify
// a buffer held in dest.
func (r *SqliteJsRows) Next(dest []driver.Value) error {
	bridge := js.Global().Get("bridge")
	res := bridge.Call("next", r.s.js)
	if res.Type() == js.TypeNull {
		return fmt.Errorf("couldn't get next row of stmt result")
	}
	for i := 0; i <= res.Length(); i++ {
		// convert from js.Value to sql.Value here
		dest[i] = res.Get(string(i))
	}
	return nil
}

// Close closes the rows iterator.
func (r *SqliteJsRows) Close() error {
	return nil
}


// Results


// LastInsertId return last inserted ID.
func (r *SqliteJsResult) LastInsertId() (int64, error) {
	return r.id, nil
}

// RowsAffected return how many rows affected.
func (r *SqliteJsResult) RowsAffected() (int64, error) {
	return r.changes, nil
}
