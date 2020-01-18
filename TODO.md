* implement transactions (begin/commit/rollback)
* implement contexts (need to delegate ExecContext, QueryContext to a goroutine and cancel it when the context is Done)
* turn main.go into a proper UT
* test transactions
* test more types
