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

// Derived from https://github.com/mattn/go-sqlite3

package sqlite3_js //nolint:golint

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"syscall/js"
)

func init() {
	sql.Register("sqlite3", &SqliteJsDriver{})
	dbMap := js.Global().Get("Map").New()
	jsEnsureGlobal(globalSQLDBs, &dbMap)
	exists := jsEnsureGlobal(globalSQLJS, nil)
	if !exists {
		panic(globalSQLJS + " must be set a global variable in JS")
	}
}

// SqliteJsDriver implements driver.Driver.
type SqliteJsDriver struct {
	ConnectHook func(*SqliteJsConn) error
}

// SqliteJsTx implements driver.Tx.
type SqliteJsTx struct {
	c *SqliteJsConn
}

// SqliteJsResult implements sql.Result.
type SqliteJsResult struct {
	js      js.Value
	id      int64
	changes int64
}

// SqliteJsRows implements driver.Rows.
type SqliteJsRows struct {
	s *SqliteJsStmt
	// nc       int
	// cols     []string
	// decltype []string
	closed bool
	cls    bool
	ctx    context.Context // no better alternative to pass context into Next() method
}

// Open a database "connection" to a SQLite database.
func (d *SqliteJsDriver) Open(dsn string) (conn driver.Conn, err error) {
	dsn = strings.TrimPrefix(dsn, "file:")
	defer protect("Open", func(e error) { err = e })
	dbMap := js.Global().Get(globalSQLDBs)
	jsDb := dbMap.Call("get", dsn)
	if !jsDb.Truthy() {
		jsDb = js.Global().Get(globalSQLJS).Get("Database").New(dsn)
		dbMap.Call("set", dsn, jsDb)
	}
	fmt.Println("Open ->", dsn, "err=", err)
	return &SqliteJsConn{
		JsDb: jsDb,
		mu:   &sync.Mutex{},
	}, nil
}

// Commit commits the transaction.
func (tx *SqliteJsTx) Commit() error {
	return nil
	/*
		if tx.c.disableTxns {
			fmt.Println("Ignoring COMMIT, txns disabled")
			return nil
		}
		_, err := tx.c.exec(context.Background(), "COMMIT", nil)
		if err != nil {
			// FIXME: ideally should only be called when
			// && err.(Error).Code == C.SQLITE_BUSY
			//
			// sqlite3 will leave the transaction open in this scenario.
			// However, database/sql considers the transaction complete once we
			// return from Commit() - we must clean up to honour its semantics.
			tx.c.exec(context.Background(), "ROLLBACK", nil)
		}
		return err */
}

// Rollback aborts the transaction.
func (tx *SqliteJsTx) Rollback() error {
	return nil
	/*
		if tx.c.disableTxns {
			fmt.Println("Ignoring ROLLBACK, txns disabled")
			return nil
		}
		_, err := tx.c.exec(context.Background(), "ROLLBACK", nil)
		return err */
}

// Rows

// Columns returns the names of the columns. The number of
// columns of the result is inferred from the length of the
// slice. If a particular column name isn't known, an empty
// string should be returned for that entry.
func (r *SqliteJsRows) Columns() []string {
	res := r.s.js.Call("getColumnNames")
	cols := make([]string, res.Length())
	for i := 0; i < res.Length(); i++ {
		cols[i] = res.Get(strconv.Itoa(i)).String()
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
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	if r.s.closed {
		return io.EOF
	}

	if r.ctx.Done() == nil {
		return r.nextSyncLocked(dest)
	}
	resultCh := make(chan error)
	go func() {
		defer func() {
			if perr := recover(); perr != nil {
				fmt.Printf("SqliteJsRows.Next panicked! err=%s", perr)
				resultCh <- fmt.Errorf("SqliteJsRows.Next panicked! err=%s", perr)
			}
		}()
		resultCh <- r.nextSyncLocked(dest)
	}()
	select {
	case err := <-resultCh:
		return err
	case <-r.ctx.Done():
		select {
		case <-resultCh: // no need to interrupt
		default:
			// this is still racy and can be no-op if executed between sqlite3_* calls in nextSyncLocked.
			// FIXME: find a way to interrupt
			// C.sqlite3_interrupt(rc.s.c.db)
			<-resultCh // ensure goroutine completed
		}
		return r.ctx.Err()
	}
}

// nextSyncLocked moves cursor to next; must be called with locked mutex.
func (r *SqliteJsRows) nextSyncLocked(dest []driver.Value) error {
	rr := r.s.Next()
	if rr == nil {
		return io.EOF
	}
	res := *rr
	for i := 0; i < res.Length(); i++ {
		jsVal := res.Get(strconv.Itoa(i))
		switch t := jsVal.Type(); t {
		case js.TypeNull:
			dest[i] = nil
		case js.TypeBoolean:
			dest[i] = jsVal.Bool()
		case js.TypeNumber:
			dest[i] = jsVal.Int()
		case js.TypeString:
			dest[i] = jsVal.String()
		case js.TypeSymbol:
			log.Fatal("Don't know how to handle Symbols yet")
		case js.TypeObject:
			// check for []byte
			if jsVal.Get("byteLength").Truthy() {
				uint8slice := make([]uint8, jsVal.Get("byteLength").Int())
				js.CopyBytesToGo(uint8slice, jsVal)
				dest[i] = uint8slice
			} else {
				log.Fatal("Don't know how to handle Objects yet")
			}
		case js.TypeFunction:
			log.Fatal("Don't know how to handle Functions yet")
		}
	}
	return nil
}

// Close closes the rows iterator.
func (r *SqliteJsRows) Close() error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	if r.s.closed || r.closed {
		return nil
	}
	r.closed = true
	if r.cls {
		return r.s.Close()
	}

	r.s.js.Call("reset")
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
