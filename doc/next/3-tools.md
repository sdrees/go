## Tools {#tools}

### Go command {#go-command}

### Cgo {#cgo}

Cgo currently refuses to compile calls to a C function which has multiple
incompatible declarations. For instance, if `f` is declared as both `void f(int)`
and `void f(double)`, cgo will report an error instead of possibly generating an
incorrect call sequence for `f(0)`. New in this release is a better detector for
this error condition when the incompatible declarations appear in different
files. See [#67699](/issue/67699).

### Vet

The new `tests` analyzer reports common mistakes in declarations of
tests, fuzzers, benchmarks, and examples in test packages, such as
malformed names, incorrect signatures, or examples that document
non-existent identifiers. Some of these mistakes may cause tests not
to run.

This analyzer is among the subset of analyzers that are run by `go test`.

### GOCACHEPROG

The `cmd/go` internal binary and test caching mechanism can now be implemented
by child processes implementing a JSON protocol between the `cmd/go` tool
and the child process named by the `GOCACHEPROG` environment variable.
This was previously behind a GOEXPERIMENT.
For protocol details, see [#59719](/issue/59719).
