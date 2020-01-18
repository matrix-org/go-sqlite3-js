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

import "database/sql"
import "log"

var c chan struct{}

func init() {
    c = make(chan struct{})
}

func main() {
    var db *sql.DB
    var err error
    if db, err = sql.Open("sqlite-js", "test.db"); err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec("create table foo(id int)")
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec("insert into foo values(42)")
    if err != nil {
        log.Fatal(err)
    }

    _, err = db.Exec("insert into foo values(?)", 31337)
    if err != nil {
        log.Fatal(err)
    }

    var stmt *sql.Stmt
    if stmt, err = db.Prepare("insert into foo values(?)"); err != nil {
        log.Fatal(err)
    }
    stmt.Exec(12345678)

    if stmt, err = db.Prepare("select * from foo"); err != nil {
        log.Fatal(err)
    }
    var id int
    stmt.QueryRow().Scan(&id)
    log.Printf("Got first id: %d", id)
    stmt.Close()

    rows, err := db.Query("select * from foo")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    for rows.Next() {
        err := rows.Scan(&id)
        if err != nil {
            log.Fatal(err)
        }
        log.Printf("Got id: %d", id)
    }
    
    <-c
}

