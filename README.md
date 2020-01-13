### go-sqlite3-js

Experimental SQL driver for sql.js (in-browser sqlite) from Go WASM.

Only implements the subset of the SQL API required by Dendrite.

To run:

```bash
# for go:
(cd go; GOOS=js GOARCH=wasm go build -o ../main.wasm)
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .

# for js:
yarn install
yarn run start
```
