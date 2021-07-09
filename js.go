package sqlite3_js //nolint:golint

import (
	"fmt"
	"os"
	"runtime/debug"
	"syscall/js"
)

const (
	// The name of the global where sql.js has been loaded. This is the `SQL` var of:
	//     const initSqlJs = require('sql.js');
	//     const SQL = await initSqlJs({ ...})
	globalSQLJS = "_go_sqlite"

	// The name of the global where sql.js should store its databases. This is purely
	// for debugging as JS-land doesn't ever read this map, but Go stores databases here.
	// The value of this global is an empty Map. It is crucial this isn't an empty object
	// else database names like 'hasOwnProperty' will fail due to it existing but
	// not being a database object!
	globalSQLDBs = "_go_sqlite_dbs"
)

// jsEnsureGlobal is a helper function to set-if-not-exists and return whether the global existed.
func jsEnsureGlobal(globalName string, defaultVal *js.Value) (existed bool) {
	v := js.Global().Get(globalName)
	if v.Truthy() {
		return true
	}
	if defaultVal != nil {
		js.Global().Set(globalName, *defaultVal)
		v = *defaultVal
	}
	return false
}

// jsTryCatch is a helper function that catches exceptions/panics thrown by fn and returns them as error.
// This is useful for calling JS functions which can throw.
func jsTryCatch(fn func() js.Value) (val js.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("exception: %s", e)
		}
	}()
	return fn(), nil
}

// protect is a helper function which guards against panics, setting an error when it happens.
func protect(name string, setError func(error)) {
	err := recover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s panicked: %s\n", name, err)
		debug.PrintStack()
		setError(fmt.Errorf("%s panicked: %s", name, err))
	}
}
