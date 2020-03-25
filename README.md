### go-sqlite3-js

Experimental SQL driver for sql.js (in-browser sqlite) from Go WASM.

Only implements the subset of the SQL API required by Dendrite.

To run tests in Docker and Node:
```
$ docker build -t gsj .
$ docker run gsj
```

To run tests locally:

```bash
$ yarn install
$ GOOS=js GOARCH=wasm go test -exec="./go_sqlite_js_wasm_exec" .
```
