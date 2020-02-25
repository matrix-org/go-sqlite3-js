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

func main() {
	log.Printf("Opening sqlite3_js driver...")
	var db *sql.DB
	var err error
	if db, err = sql.Open("sqlite3_js", "test.db"); err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE bar(id INTEGER); create table foo(id INTEGER PRIMARY KEY, name string)")
	if err != nil {
		log.Fatal(err)
	}

	earlierPrepStmt, err := db.Prepare("SELECT id, name FROM foo")
	if err != nil {
		log.Fatal(err)
	}

	res, err := db.Exec("insert into bar values(9001)")
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

	rows, err = earlierPrepStmt.Query()
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		err := rows.Scan(&id, &name)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Empty db.Query() got row: %d, %s", id, name)
	}
	rows.Close()

	<-c
}
