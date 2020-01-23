* check that rows & stmts get closed up properly (the `cls` field etc)
* hook up Result{} properly
* turn main.go into a proper UT
* test more types


Stuff to check
 * That error codes get handled sensibly
 * That exceptions get trapped sensibly
 * We don't leak objects
 * We recover fromnetwork failures (if any)
