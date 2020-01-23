* implement transactions (begin/commit/rollback)
* implement contexts (need to delegate ExecContext, QueryContext to a goroutine and cancel it when the context is Done)
* figure out if we need a stmt mutex on things like Columns()
* check that rows & stmts get closed up properly (the `cls` field etc)
* hook up Result{} properly
* turn main.go into a proper UT
* test transactions
* test more types


Stuff to check
 * Exceptions don't kill everything
 * We don't leak objects
 * We recover fromnetwork failures (if any)
