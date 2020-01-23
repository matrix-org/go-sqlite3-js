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
                console.debug(`preparing query: ${query}`)
                const stmt = db.prepare(query)
                console.debug(`prepared query: ${query} as ${stmt.jb}`)
                return stmt
            },
            exec: (stmt, ...args) => {
                console.debug(`executing statement ${stmt.jb} with '${args}'`)
                return stmt.run(args)
            },
            query: (stmt, ...args) => {
                console.debug(`querying statement ${stmt.jb} with '${args}'`)
                const res = stmt.bind(args)
                if (!res) return res
                // FIXME: storing random state on stmt is horrific
                stmt._has_next = stmt.step()
                return stmt._has_next
            },
            columns: (stmt) => {
                console.debug(`getting columns as '${stmt.getColumnNames()}' from statement ${stmt.jb}`)
                return stmt.getColumnNames()
            },
            next: (stmt) => {
                if (stmt._has_next) {
                    console.debug(`getting row from statement ${stmt.jb}`)
                    const row = stmt.get()
                    // FIXME: ugly hack - surely we shouldn't have to monkey-patch this
                    stmt._has_next = stmt.step()
                    return row
                }
                else {
                    return null
                }
            },
            close: (stmt) => {
                console.debug(`freeing statement ${stmt.jb}`)
                return stmt.free()
            },
            reset: (stmt) => {
                console.debug(`resetting statement ${stmt.jb}`)
                return stmt.reset()
            }
        }
    })
}
