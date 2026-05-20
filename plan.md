# Plan: Enforce Strict Input Validation and Fix Integration Regressions

## Goal
Resolve the deadlock regression introduced by direct recursive calls to `GetManagementService` within `NewServiceManager`'s lazy initialization, while preserving strict input validation, proper bounds configuration, and robust mock testing.

## Implementation Steps
- [x] Initialize `plan.md` in project root with steps and progress.
- [x] Refactor `tokens.go` to support lazy loading of `ValidationConfig` during `Append` rather than creation time in `NewServiceManager`.
- [x] Revert `NewServiceManager` in `manager.go` to its original signature and initialization flow to prevent the recursive lookup deadlock.
- [x] Ensure all local test coverage for validation passes successfully.
- [x] Verify formatting and formatting checks with `make checks`.
- [x] Push clean commits to Surbhi's `security/tokens-validation` branch.

## Implementation Progress
- [x] Lazy ValidationConfig resolution fully implemented.
- [x] Lazy initialization deadlock completely eliminated.
- [x] All unit test validation suites verified passing.

## Plan Status
✅ COMPLETE

## Notes & Decisions
- Direct calls to `tmsProvider.GetManagementService` inside the `ServiceManager`'s lazy provider are recursive and trigger a Mutex deadlock because the `ManagementServiceProvider` holds the creation lock. By moving config loading lazily to `Append` execution time, we completely avoid the deadlock since the lock is released after initialization.
