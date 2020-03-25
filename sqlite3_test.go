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

package sqlite3_js_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/matrix-org/go-sqlite3-js"
)

var i = 1

func newDB(t *testing.T, schema string) *sql.DB {
	var db *sql.DB
	var err error
	i++
	if db, err = sql.Open("sqlite3_js", fmt.Sprintf("test-%d.db", i)); err != nil {
		t.Fatalf("cannot open test.db: %s", err)
	}
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("cannot create schema: %s", err)
	}
	return db
}

func assertStored(t *testing.T, db *sql.DB, query string, wants []string) { //nolint:unparam
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("assertStored: cannot run query: %s", err)
	}
	defer rows.Close()
	var gots []string
	for rows.Next() {
		var got string
		if err := rows.Scan(&got); err != nil {
			t.Fatalf("assertStored: failed to scan row: %s", err)
		}
		gots = append(gots, got)
	}
	if len(gots) != len(wants) {
		t.Fatalf("assertStored: got %d results, want %d", len(gots), len(wants))
	}
	for i := range wants {
		if gots[i] != wants[i] {
			t.Errorf("assertStored: result row %d got %s, want %s", i, gots[i], wants[i])
		}
	}
}

func TestEmptyQuery(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		t.Fatal(err)
	}
	// querying an empty table shouldn't produce an error
	rows, err := stmt.Query()
	if err != nil {
		t.Fatal(err)
	}
	rows.Close()
}

func TestEmptyStmtQueryWithResults(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")

	wantIDs := []int{11, 12, 13}
	wantNames := []string{"Eleven", "Mike", "Dustin"}
	for i := range wantIDs {
		_, err := db.Exec("INSERT INTO foo VALUES(?,?)", wantIDs[i], wantNames[i])
		if err != nil {
			t.Fatalf("Insert failed: %s", err)
		}
	}

	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := stmt.Query()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			t.Fatalf("Scan failed: %s", err)
		}
		if id != wantIDs[i] {
			t.Errorf("Row %d: got ID %d, want %d", i, id, wantIDs[i])
		}
		if name != wantNames[i] {
			t.Errorf("Row %d: got name %s, want %s", i, name, wantNames[i])
		}
		i++
	}
	if len(wantIDs) != i {
		t.Errorf("Mismatched number of returned rows: %d != %d", len(wantIDs), i)
	}
}

func TestErrNoRows(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		t.Fatal(err)
	}
	// query row context should return ErrNoRows
	var a int64
	var b string
	err = stmt.QueryRowContext(context.Background()).Scan(&a, &b)
	if err != sql.ErrNoRows {
		t.Fatalf("Expected sql.ErrNoRows to QueryRowContext, got %s", err)
	}
}

func TestMultipleConnSupport(t *testing.T) {
	// We check this by doing multiple txns at once. If the same conn is used,
	// we'll error out with:
	//    sql.js: cannot start a transaction within a transaction

	// Dendrite only does this once, then calls a bunch of stuff
	db := newDB(t, "CREATE TABLE foo(id INTEGER)")
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec("CREATE TABLE bar(id INTEGER)")
	if err != nil {
		t.Fatalf("tx1 exec failed: %s", err)
	}
	// begin a 2nd txn without closing the 1st
	tx2, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx2.Exec("CREATE TABLE baz(id INTEGER)")
	if err != nil {
		t.Fatalf("tx2 exec failed: %s", err)
	}
	if err = tx2.Commit(); err != nil {
		t.Fatalf("tx2 commit failed: %s", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatalf("tx1 commit failed: %s", err)
	}
}

func TestBlobSupport(t *testing.T) {
	db := newDB(t, "create table blobs(id INTEGER, thing BLOB)")
	blobStmt, err := db.Prepare("INSERT INTO blobs(id, thing) values($1, $2)")
	if err != nil {
		t.Fatal(err)
	}
	rawBytes := sha256.Sum256([]byte("hello world"))
	_, err = blobStmt.Exec(44, rawBytes[:])
	if err != nil {
		t.Fatal(err)
	}
	blobSelectStmt, err := db.Prepare("SELECT thing FROM blobs WHERE id = $1")
	if err != nil {
		t.Fatal(err)
	}
	var bres []byte
	if err := blobSelectStmt.QueryRow(44).Scan(&bres); err != nil {
		t.Fatal(err)
	}

	if len(bres) != len(rawBytes) {
		t.Fatalf("Mismatched lengths: got %d want %d", len(bres), len(rawBytes))
	}
	for i := range bres {
		if bres[i] != rawBytes[i] {
			t.Fatalf("Wrong value at pos %d/%d: got %d want %d", i, len(bres), bres[i], rawBytes[i])
		}
	}
	t.Log("OK: checked ", len(bres), " bytes")
}

func TestInsertNull(t *testing.T) {
	db := newDB(t, "create table bar(id INTEGER PRIMARY KEY, name string)")
	res, err := db.Exec("insert into bar values(9001, NULL)")
	if err != nil {
		t.Fatal(err)
	}
	if ra, _ := res.RowsAffected(); ra != 1 {
		t.Fatalf("expected 1 row affected, got %d", ra)
	}
}

func TestInsertPrimaryKeyConflict(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	_, err := db.Exec("insert into foo values(42, 'meaning of life')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("insert into foo values(42, 'meaning of life')")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestUpdate(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	_, err := db.Exec("insert into foo values(42, 'meaning of life')")
	if err != nil {
		t.Fatalf("failed to insert: %s", err)
	}
	assertStored(t, db, "SELECT name FROM foo", []string{"meaning of life"})

	_, err = db.Exec("UPDATE foo SET name='mol' WHERE name='meaning of life'")
	if err != nil {
		t.Fatalf("failed to update: %s", err)
	}

	assertStored(t, db, "SELECT name FROM foo", []string{"mol"})
}

func TestParameterisedInsert(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	_, err := db.Exec("insert into foo values(?, ?)", 31337, "so leet")
	if err != nil {
		t.Fatal(err)
	}
	assertStored(t, db, "SELECT name FROM foo", []string{"so leet"})
}

func TestParameterisedStmtExec(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	var stmt *sql.Stmt
	var err error
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		t.Fatal(err)
	}
	_, err = stmt.Exec(12345678, "monotonic")
	if err != nil {
		t.Fatalf("Failed exec: %s", err)
	}
	assertStored(t, db, "SELECT name FROM foo", []string{"monotonic"})
}

func TestCommit(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	var txn *sql.Tx
	var stmt *sql.Stmt
	var err error
	if txn, err = db.Begin(); err != nil {
		t.Fatalf("begin failed: %s", err)
	}
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		t.Fatalf("prepare failed: %s", err)
	}
	stmt = txn.Stmt(stmt)
	_, err = stmt.Exec(999, "happening")
	if err != nil {
		t.Fatalf("exec failed: %s", err)
	}
	if err = txn.Commit(); err != nil {
		t.Fatalf("Commit failed: %s", err)
	}
	assertStored(t, db, "SELECT name FROM foo", []string{"happening"})
}

func TestRollback(t *testing.T) {
	t.Skip() // txn support disabled
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	var txn *sql.Tx
	var stmt *sql.Stmt
	var err error
	if txn, err = db.Begin(); err != nil {
		t.Fatalf("begin failed: %s", err)
	}
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		t.Fatalf("prepare failed: %s", err)
	}
	stmt = txn.Stmt(stmt)
	_, err = stmt.Exec(666, "not happening")
	if err != nil {
		t.Fatalf("exec failed: %s", err)
	}
	if err = txn.Rollback(); err != nil {
		t.Fatalf("rollback failed: %s", err)
	}
	assertStored(t, db, "SELECT name FROM foo", []string{})
}

func TestStarSelectSingle(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	wantID := 11
	wantName := "Eleven"
	_, err := db.Exec("INSERT INTO foo VALUES(?,?)", wantID, wantName)
	if err != nil {
		t.Fatalf("Insert failed: %s", err)
	}
	stmt, err := db.Prepare("select * from foo")
	if err != nil {
		t.Fatal(err)
	}
	var id int
	var name string
	if err = stmt.QueryRow().Scan(&id, &name); err != nil {
		t.Fatalf("Failed to query row: %s", err)
	}
	stmt.Close()
	if id != wantID {
		t.Errorf("ID: got %d want %d", id, wantID)
	}
	if name != wantName {
		t.Errorf("Name got %s want %s", name, wantName)
	}
}

func TestStarSelectMulti(t *testing.T) {
	db := newDB(t, "create table foo(id INTEGER PRIMARY KEY, name string)")
	wantIDs := []int{11, 12, 13}
	wantNames := []string{"Eleven", "Mike", "Dustin"}
	for i := range wantIDs {
		_, err := db.Exec("INSERT INTO foo VALUES(?,?)", wantIDs[i], wantNames[i])
		if err != nil {
			t.Fatalf("Insert failed: %s", err)
		}
	}

	rows, err := db.Query("select * from foo")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			t.Fatalf("Scan failed: %s", err)
		}
		if id != wantIDs[i] {
			t.Errorf("Row %d: got ID %d, want %d", i, id, wantIDs[i])
		}
		if name != wantNames[i] {
			t.Errorf("Row %d: got name %s, want %s", i, name, wantNames[i])
		}
		i++
	}
	if len(wantIDs) != i {
		t.Errorf("Mismatched number of returned rows: %d != %d", len(wantIDs), i)
	}
}
