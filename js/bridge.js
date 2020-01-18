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

    global.bridge = {
        open: (dsn) => {
            return new SQL.Database()
        },
        prepare: (db, query) => {
            return db.prepare(query)
        },
        exec: (stmt, ...args) => {
            return stmt.exec(args)
        },
        close: (stmt) => {
            return stmt.free()
        },
    }

    const go = new Go()
    WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
        go.run(result.instance)
    });
});

