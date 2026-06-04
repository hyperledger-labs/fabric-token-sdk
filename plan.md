# Plan: Fix Blocking Channel Receives Without Timeout or Context Cancellation

## Goal
Remove or document indefinite blocking in production and test code paths where channel receives can wait forever without a timeout or context cancellation.

## Root Cause
Several code paths wait on channels with a plain `<-ch` receive. If the sender never writes or the goroutine never exits, these code paths can block indefinitely and hang the process or test.

## Affected Files
- `token/services/selector/simple/inmemory/locker.go`
- `token/services/selector/sherdlock/manager.go`
- `token/services/ttx/boolpolicy/spend.go`
- `token/services/ttx/multisig/spend.go`
- `token/services/network/fabricx/finality/queue/queue.go`
- `integration/token/fungible/tests.go`
- `integration/token/common/views/finality.go`
- `integration/nwo/txgen/service/runner/base_runner.go`

## Proposed Fixes
1. `locker.go`: add a bounded wait in `Stop()` so it does not block forever if scan goroutine fails to exit.
2. `manager.go`: add a bounded wait in `Stop()` for cleaner shutdown.
3. `boolpolicy/spend.go`: prevent indefinite receive from `answerChannel` by adding a request timeout, with a new `WithTimeout` option.
4. `multisig/spend.go`: implement the existing timeout handling for answer receives.
5. `queue.go`: document that `Shutdown(timeout==0)` waits forever and preserve the explicit semantics.
6. `integration/token/fungible/tests.go`: add receive timeouts to transient goroutine result channels.
7. `integration/token/common/views/finality.go`: add a bounded wait for the second finality error receive and enforce a default timeout when zero is supplied.
8. `base_runner.go`: add a bounded wait during `ShutDown()` to avoid indefinite blocking.

## Implementation Steps
1. Add a `stopWaitTimeout` constant and use `select` with `time.After` in `locker.Stop()` and `manager.Stop()`.
2. Update `RequestSpendView` in `boolpolicy` with a `timeout time.Duration` field, `WithTimeout()` setter, and a guarded receive on `answerChannel`.
3. Update `RequestSpendView` in `multisig` to use `c.timeout` for channel receive timeouts.
4. Add a comment in `queue.go` clarifying `timeout==0` means wait forever.
5. Add per-channel receive timeout handling in `integration/token/fungible/tests.go` loops.
6. Update `integration/token/common/views/finality.go` to use a default timeout when none is specified and handle channel receives with `select`.
7. Add a timeout to `BaseRunner.ShutDown()` and return an error if shutdown does not complete.
8. Run targeted tests for changed packages and packages with related shutdown behavior.

## Test Plan
- Run `go test ./token/services/selector/simple/inmemory ./token/services/selector/sherdlock ./token/services/ttx/boolpolicy ./token/services/ttx/multisig ./token/services/network/fabricx/finality ./integration/token/common ./integration/nwo/txgen/service/runner`
- If integration packages are too heavy, run compile-only checks for modified files and existing unit tests.
- Verify `go test ./integration/token/fungible` if environment permits.

## Risks
- Changing shutdown semantics in `queue.go` could alter behavior if callers rely on zero meaning "wait forever." We'll preserve semantics with a comment.
- Adding default timeout in `finality.go` can change view behavior for callers that pass `Timeout == 0`; choose a large default and document it.
- New timeouts must be long enough for normal operation but short enough to catch hangs.

## Implementation Progress
- [ ] Review and update affected production code
- [ ] Update tests and add regression coverage
- [ ] Run targeted tests and verify
- [ ] Final review and cleanup
