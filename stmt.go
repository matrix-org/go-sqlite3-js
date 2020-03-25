package sqlite3_js //nolint:golint

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sync"
	"syscall/js"
)

// SqliteJsStmt implements driver.Stmt.
type SqliteJsStmt struct {
	c       *SqliteJsConn
	js      js.Value // sql.js Statement: https://sql-js.github.io/sql.js/documentation/class/Statement.html
	mu      sync.Mutex
	closed  bool
	cls     bool // wild guess: connection level statement?
	hasNext bool
}

type namedValue struct {
	Name    string
	Ordinal int
	Value   driver.Value
}

// Exec executes a prepared statement with the given arguments and returns a
// Result summarizing the effect of the statement.
func (s *SqliteJsStmt) Exec(args []driver.Value) (driver.Result, error) {
	list := make([]namedValue, len(args))
	for i, v := range args {
		list[i] = namedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return s.exec(context.Background(), list)
}

// ExecContext executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
func (s *SqliteJsStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	list := make([]namedValue, len(args))
	for i, nv := range args {
		list[i] = namedValue(nv)
	}
	return s.exec(ctx, list)
}

// exec executes a query that doesn't return rows. Attempts to honor context timeout.
func (s *SqliteJsStmt) exec(ctx context.Context, args []namedValue) (driver.Result, error) {
	if ctx.Done() == nil {
		return s.execSync(args)
	}

	type result struct {
		r   driver.Result
		err error
	}
	resultCh := make(chan result)
	go func() {
		defer func() {
			if perr := recover(); perr != nil {
				fmt.Printf("SqliteJsStmt.exec panicked! nargs=%d err=%s", len(args), perr)
				resultCh <- result{nil, fmt.Errorf("SqliteJsStmt.exec panicked! nargs=%d err=%s", len(args), perr)}
			}
		}()
		r, err := s.execSync(args)
		resultCh <- result{r, err}
	}()
	select {
	case rv := <-resultCh:
		return rv.r, rv.err
	case <-ctx.Done():
		select {
		case <-resultCh: // no need to interrupt
		default:
			// FIXME: find a way to actually interrupt the connection
			// this is still racy and can be no-op if executed between sqlite3_* calls in execSync.
			// C.sqlite3_interrupt(s.c.db)
			<-resultCh // ensure goroutine completed
		}
		return nil, ctx.Err()
	}
}

func (s *SqliteJsStmt) execSync(args []namedValue) (driver.Result, error) {
	// We're going to issue a bunch of JS calls, some of which (last rowid)
	// are NOT statement-level scoped, but connection-level scoped, so we cannot just
	// lock the statement mutex we have already, otherwise multiple goroutines may
	// exec in this function, causing the last insert rowid to be wrong.
	s.c.mu.Lock()
	defer s.c.mu.Unlock()

	jsArgs := make([]interface{}, len(args))
	for i, v := range args {
		if bval, ok := v.Value.([]byte); ok {
			dst := js.Global().Get("Uint8Array").New(len(bval))
			js.CopyBytesToJS(dst, bval)
			jsArgs[i] = dst
		} else {
			jsArgs[i] = js.ValueOf(v.Value)
		}
	}
	result, err := jsTryCatch(func() js.Value { return s.js.Call("run", jsArgs) })
	if err != nil {
		return nil, fmt.Errorf("execSync sql.js: %s", err)
	}

	// TODO: Kinda sucks each exec is paired with 2 extra calls but we have to do it ASAP else we risk
	// getting out of sync with subsequent inserts.
	rowsModified := s.c.JsDb.Call("getRowsModified")

	rowidRes, err := jsTryCatch(func() js.Value {
		rows := s.c.JsDb.Call("exec", "SELECT last_insert_rowid()")
		if rows.Length() != 1 { // query result
			// this gets recover()d and turns into an error
			panic(fmt.Sprintf("last_insert_rowid: expected 1 row to be returned, got %d", rows.Length()))
		}
		// 'rows' is of the form: [{columns: ['id'], values:[[1],[2],[3]]}]
		return rows.Index(0).Get("values").Index(0).Index(0)
	})
	if err != nil {
		return nil, fmt.Errorf("execSync: error getting rowid: %s", err)
	}
	return &SqliteJsResult{
		js:      result,
		changes: int64(rowsModified.Int()),
		id:      int64(rowidRes.Int()),
	}, nil
}

// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or additionally).
func (s *SqliteJsStmt) Query(args []driver.Value) (driver.Rows, error) {
	list := make([]namedValue, len(args))
	for i, v := range args {
		list[i] = namedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}
	return s.query(context.Background(), list)
}

// QueryContext executes a query that may return rows, such as a
// SELECT.
//
// QueryContext must honor the context timeout and return when it is canceled.
func (s *SqliteJsStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	list := make([]namedValue, len(args))
	for i, nv := range args {
		list[i] = namedValue(nv)
	}
	return s.query(ctx, list)
}

func (s *SqliteJsStmt) query(ctx context.Context, args []namedValue) (driver.Rows, error) {
	jsArgs := make([]interface{}, len(args))
	for i, v := range args {
		if bval, ok := v.Value.([]byte); ok {
			dst := js.Global().Get("Uint8Array").New(len(bval))
			js.CopyBytesToJS(dst, bval)
			jsArgs[i] = dst
		} else {
			jsArgs[i] = js.ValueOf(v.Value)
		}
	}
	jsOk := s.js.Call("bind", jsArgs)
	if !jsOk.Bool() {
		return nil, fmt.Errorf("failed to bind query")
	}
	res := s.js.Call("step")
	s.hasNext = res.Bool()

	return &SqliteJsRows{
		s:   s,
		cls: s.cls, // FIXME: we never set s.cls, as we haven't implemented conn.Query(), which would set it
		ctx: ctx,
	}, nil
}

func (s *SqliteJsStmt) Next() *js.Value {
	if !s.hasNext {
		return nil
	}
	row := s.js.Call("get")
	jsHasNext := s.js.Call("step")
	s.hasNext = jsHasNext.Bool()
	return &row
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (s *SqliteJsStmt) NumInput() int {
	return -1
}

// Close closes the statement.
func (s *SqliteJsStmt) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	res := s.js.Call("free")
	if !res.Bool() {
		return fmt.Errorf("couldn't close stmt")
	}
	return nil
}
