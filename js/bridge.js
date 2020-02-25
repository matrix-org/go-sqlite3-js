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

import initSqlJs from 'sql.js'

global._go_sqlite_dbs = {}

// exposes sql.js's API in a shape that looks more like a go sql driver.
// the datatype conversions however happens on the go side, rather than here.

export function init(config) {
    return initSqlJs(config).then(SQL => {
        global._go_sqlite_bridge = {
            open: (dsn) => {
                if (global._go_sqlite_dbs[dsn]) {
                    console.debug(`re-opening db ${dsn}`)
                    return global._go_sqlite_dbs[dsn]
                }
                else {
                    console.debug(`opening db ${dsn}`)
                    const db = new SQL.Database()
                    global._go_sqlite_dbs[dsn] = db
                    return db
                }
            },
            prepare: (db, query) => {
                const stmt = db.prepare(query)
                console.debug(`db.prepare: ${stmt.jb} => ${query}`)
                return stmt
            },
            execMany: (db, query) => {
                let res = {
                    result: null,
                    error: null,
                }
                try {
                    console.debug(`db.exec: ${query}`)
                    res.result = db.exec(query);
                } catch (err) {
                    res.error = err;
                }
                return res;
            },
            exec: (stmt, ...args) => {
                console.debug(`stmt.run: ${stmt.jb} => '${args}'`)
                let retres = null;
                let reterr = null;
                try {
                    const res = stmt.run(args)
                    retres = res
                } catch (err) {
                    reterr = err;
                }
                return {
                    result: retres,
                    error: reterr,
                };
            },

            getRowsModified: (db) => {
                return db.getRowsModified()
            },
            lastInsertRowid: (db) => {
                let reterr = null;
                let retres = null;
                try {
                    const res = db.exec("SELECT last_insert_rowid()") // this defaults to the most recent table hence this works
                    if (res.length !== 1) { // query result
                        reterr = new Error("last_insert_rowid returned no result");
                    }
                    // results of form:
                    // {columns: ['id'], values:[[1],[2],[3]]},
                    retres = res[0]["values"][0][0];
                } catch (err) {
                    reterr = err;
                }
                return {
                    result: retres,
                    error: reterr,
                };
            },
            query: (stmt, ...args) => {
                console.debug(`stmt.bind: ${stmt.jb} => '${args}'`)
                const res = stmt.bind(args)
                if (!res) return res
                // FIXME: storing random state on stmt is horrific
                console.debug(`stmt.step: ${stmt.jb}`)
                stmt._has_next = stmt.step()
                return stmt._has_next
            },
            columns: (stmt) => {
                console.debug(`stmt.getColumnNames(): ${stmt.jb} => '${stmt.getColumnNames()}'`)
                return stmt.getColumnNames()
            },
            next: (stmt) => {
                if (stmt._has_next) {
                    console.debug(`next => stmt.get: ${stmt.jb}`)
                    const row = stmt.get()
                    // FIXME: ugly hack - surely we shouldn't have to monkey-patch this
                    console.debug(`next => stmt.step: ${stmt.jb}`)
                    stmt._has_next = stmt.step()
                    return row
                }
                else {
                    return null
                }
            },
            close: (stmt) => {
                console.debug(`stmt.free: ${stmt.jb}`)
                return stmt.free()
            },
            reset: (stmt) => {
                console.debug(`stmt.reset: ${stmt.jb}`)
                return stmt.reset()
            }
        }
    })
}
