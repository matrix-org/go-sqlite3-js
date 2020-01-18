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


initSqlJs().then(SQL => {

    // exposes sql.js's API in a shape that looks more like a go sql driver.
    // the datatype conversions however happens on the go side, rather than here.
    global.bridge = {
        open: (dsn) => {
            console.log(`opening db ${dsn}`)
            global.db = new SQL.Database()
            return global.db
        },
        prepare: (db, query) => {
            console.log(`preparing query: ${query}`)
            return db.prepare(query)
        },
        exec: (stmt, ...args) => {
            console.log(`executing statement ${stmt.jb} with '${args}'`)
            return stmt.run(args)
        },
        query: (stmt, ...args) => {
            console.log(`querying statement ${stmt.jb} with '${args}'`)
            const res = stmt.bind(args)
            if (!res) return res
            return stmt.step()
        },
        columns: (stmt) => {
            console.log(`getting columns as '${stmt.getColumnNames()}' from statement ${stmt.jb}`)
            return stmt.getColumnNames()
        },
        next: (stmt) => {
            console.log(`getting row from statement ${stmt.jb}`)
            return stmt.get()
        },
        close: (stmt) => {
            console.log(`freeing statement ${stmt.jb}`)
            return stmt.free()
        },
    }

    const go = new Go()
    WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
        go.run(result.instance)
    });
});

