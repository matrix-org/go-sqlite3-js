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

package main

// GOOS=js GOARCH=wasm go build -o main.wasm  ./tests/main.go
// cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/matrix-org/go-sqlite3-js"
)

var c chan struct{}

func init() {
	c = make(chan struct{})
}

func printResultMetadata(msg string, res sql.Result) {
	if res == nil {
		log.Printf("nil result")
		return
	}
	// the impl never returns an error on these functions
	ra, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	rowid, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s => rowid: %d rows affected: %d", msg, rowid, ra)
}

func checkBlobSupport(db *sql.DB) {
	log.Println("checkBlobSupport:")
	blobStmt, err := db.Prepare("INSERT INTO bar(id, thing) values($1, $2)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = blobStmt.Exec(44, []byte("hello world"))
	if err != nil {
		log.Fatal(err)
	}
	blobSelectStmt, err := db.Prepare("SELECT thing FROM bar WHERE id = $1")
	if err != nil {
		log.Fatal(err)
	}
	var bres []byte
	if err := blobSelectStmt.QueryRow(44).Scan(&bres); err != nil {
		log.Fatal(err)
	}
	log.Printf("blob select: %s", bres)
}

func checkEmptyQuery(db *sql.DB) {
	log.Println("checkEmptyQuery:")
	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		log.Fatal(err)
	}
	// querying an empty table shouldn't produce an error
	rows, err := stmt.Query()
	if err != nil {
		log.Fatal(err)
	}
	rows.Close()
}

func checkEmptyQueryWithResults(db *sql.DB) {
	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		log.Fatal(err)
	}
	rows, err := stmt.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Empty db.Query() got row: %d, %s", id, name)
	}
}

func checkErrNoRows(db *sql.DB) {
	log.Println("checkErrNoRows:")
	stmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		log.Fatal(err)
	}
	// query row context should return ErrNoRows
	var a int64
	var b string
	err = stmt.QueryRowContext(context.Background()).Scan(&a, &b)
	if err != sql.ErrNoRows {
		log.Fatalf("Expected sql.ErrNoRows to QueryRowContext, got %s", err)
	} else {
		log.Println("Returns sql.ErrNoRows ok")
	}
}

func checkMultipleConnSupport() {
	// We check this by doing multiple txns at once. If the same conn is used,
	// we'll error out with:
	//    sql.js: cannot start a transaction within a transaction
	log.Println("checkMultipleConnSupport:")
	var db *sql.DB
	var err error
	// Dendrite only does this once, then calls a bunch of stuff
	if db, err = sql.Open("sqlite3_js", "test2.db?txns=false"); err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.Exec("CREATE TABLE foo(id INTEGER)")
	if err != nil {
		log.Fatal(err)
	}
	tx2, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx2.Exec("CREATE TABLE bar(id INTEGER)")
	if err != nil {
		log.Fatal(err)
	}
	if err = tx2.Commit(); err != nil {
		log.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Printf("Opening sqlite3_js driver...")
	var db *sql.DB
	var err error
	if db, err = sql.Open("sqlite3_js", "test.db"); err != nil {
		log.Fatal(err)
	}

	// checks multiple queries can go in a single Exec call. We don't support params in this form.
	_, err = db.Exec("CREATE TABLE bar(id INTEGER, thing BLOB); create table foo(id INTEGER PRIMARY KEY, name string)")
	if err != nil {
		log.Fatal(err)
	}

	checkBlobSupport(db)

	checkEmptyQuery(db)

	checkErrNoRows(db)

	res, err := db.Exec("insert into bar values(9001, NULL)")
	if err != nil {
		log.Fatal(err)
	}
	printResultMetadata("After insert on db", res)

	res, err = db.Exec("insert into foo values(42, 'meaning of life')")
	if err != nil {
		log.Fatal(err)
	}
	printResultMetadata("After insert on db", res)

	res, err = db.Exec("insert into foo values(42, 'meaning of life')")
	if err == nil {
		log.Fatal("Expected err returned from primary key conflict INSERT but got nil")
	}

	res, err = db.Exec("insert into foo values(43, 'meaning of life')")
	if err != nil {
		log.Fatal(err)
	}

	res, err = db.Exec("UPDATE foo SET name='mol' WHERE name='meaning of life'")
	if err != nil {
		log.Fatal(err)
	}
	printResultMetadata("After updating 2 rows", res)

	_, err = db.Exec("insert into foo values(?, ?)", 31337, "so leet")
	if err != nil {
		log.Fatal(err)
	}

	var stmt *sql.Stmt
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		log.Fatal(err)
	}
	res, err = stmt.Exec(12345678, "monotonic")
	printResultMetadata("After insert on stmt", res)

	var txn *sql.Tx
	if txn, err = db.Begin(); err != nil {
		log.Fatal(err)
	}
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		log.Fatal(err)
	}
	stmt = txn.Stmt(stmt)
	stmt.Exec(666, "not happening")
	txn.Rollback()

	if txn, err = db.Begin(); err != nil {
		log.Fatal(err)
	}
	if stmt, err = db.Prepare("insert into foo values(?, ?)"); err != nil {
		log.Fatal(err)
	}
	stmt = txn.Stmt(stmt)
	res, err = stmt.Exec(999, "happening")
	printResultMetadata("After insert on stmt in txn", res)
	txn.Commit()

	if stmt, err = db.Prepare("select * from foo"); err != nil {
		log.Fatal(err)
	}
	var id int
	var name string
	stmt.QueryRow().Scan(&id, &name)
	log.Printf("Got first row: %d, %s", id, name)
	stmt.Close()

	rows, err := db.Query("select * from foo")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Got row: %d, %s", id, name)
	}
	rows.Close()

	checkEmptyQueryWithResults(db)

	checkMultipleConnSupport()

	<-c
}
