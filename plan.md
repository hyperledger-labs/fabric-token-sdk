# Plan: Database Connection TLS Support

## Goal
Add support for TLS-secured database connections in the Token SDK by parsing options, loading certificates, and programmatically registering connection configuration with `pgx/v5/stdlib`.

## Implementation Steps
1. [ ] Create `tls.go` in `token/services/storage/db/sql/postgres/` defining `TLSConfig`, `tlsConfigProvider`, and TLS connection registration.
2. [ ] Modify `driver.go` in `token/services/storage/db/sql/postgres/` to wrap database configuration in `tlsConfigProvider`.
3. [ ] Create `tls_test.go` in `token/services/storage/db/sql/postgres/` with comprehensive unit tests for configuration parsing and connection registration.
4. [ ] Format code, run linter checks, and run unit tests in `token/services/storage/db/sql/postgres/`.

## Implementation Progress
- [x] Step 1: Done - created tls.go
- [x] Step 2: Done - wrapped base config provider in driver.go
- [x] Step 3: Done - created tls_test.go
- [/] Step 4: In progress - formatting, checking, and running tests

## Notes & Decisions
- None so far.
