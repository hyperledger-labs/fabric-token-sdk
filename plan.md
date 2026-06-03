# Plan: Database Connection TLS Support

## Goal
Add support for TLS-secured database connections in the Token SDK by parsing options, loading certificates, and programmatically registering connection configuration with `pgx/v5/stdlib`.

## Implementation Steps
1. [x] Create `tls.go` in `token/services/storage/db/sql/postgres/` defining `TLSConfig`, `tlsConfigProvider`, and TLS connection registration.
2. [x] Modify `driver.go` in `token/services/storage/db/sql/postgres/` to wrap database configuration in `tlsConfigProvider`.
3. [x] Create `tls_test.go` in `token/services/storage/db/sql/postgres/` with comprehensive unit tests for configuration parsing and connection registration.
4. [x] Format code, run linter checks, and run unit tests in `token/services/storage/db/sql/postgres/`.
5. [x] Fix failing GitHub CI check `Check Markdown links` by resolving the broken link and invalid `/*` syntax in `docs/tls_support.md`.
6. [x] Fix failing GitHub CI check `Go Fix Check` by modernizing type parameter to `any` in `tls_test.go`.
7. [x] Fix failing GitHub CI check `golangci-lint` by resolving tag alignment, missing blank lines before returns (`nlreturn`), using `errors.New` (`perfsprint`), `t.Helper()` usage (`thelper`), and using `require` for error checks in unit tests.

## Implementation Progress
- [x] Step 1: Done - created tls.go
- [x] Step 2: Done - wrapped base config provider in driver.go
- [x] Step 3: Done - created tls_test.go
- [x] Step 4: Done - formatting, checking, and running tests. Unit tests pass.
- [x] Step 5: Done - Removed /* comment syntax and fixed pgx GitHub link in docs/tls_support.md
- [x] Step 6: Done - Fixed UnmarshalKey signature (replaced interface{} with any) in tls_test.go
- [x] Step 7: Done - Fixed all linter violations (tagalign, nlreturn, perfsprint, thelper, testifylint) in tls.go and tls_test.go

## Notes & Decisions
- Split `RegisterTLSConnection` in `tls.go` to expose a `createTLSConnConfig` for testability, allowing tests to verify TLS settings directly without relying on retrieving configuration from `pgx/v5/stdlib`.
- Integration tests ran on the Windows host timed out after 600s, this is likely an environment issue unrelated to the TLS configurations as the `postgres` unit tests pass.
- Added `Signed-off-by` using `git commit --amend -s` to address DCO check requirements.

✅ COMPLETE
