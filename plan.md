# Implementation Plan - Optimize and Fix Validation Bounds for Idemix Identities

## Goal
Increase `MaxOwnerRawSize` and `MaxIssuerRawSize` from `16 * 1024` (16KB) to `256 * 1024` (256KB) across all configurations, validators, and driver files. This resolves the regression where standard `zkatdlog` integration tests (such as `update-t2`) fail because serialized Idemix identities, audit info, and output audit info naturally exceed the highly restrictive 16KB boundary during request unmarshaling and validation.

## Implementation Steps
- [x] 1. Update `MaxOwnerRawSize` and `MaxIssuerRawSize` constants in `token/driver/validator.go` to `256 * 1024`.
- [x] 2. Update default `MaxOwnerRawSize` and `MaxIssuerRawSize` values in `token/config.go` to `256 * 1024`.
- [x] 3. Update default `MaxOwnerRawSize` and `MaxIssuerRawSize` values in `token/services/tokens/tokens.go` to `256 * 1024`.
- [x] 4. Update default `MaxOwnerRawSize` and `MaxIssuerRawSize` values in `token/core/common/validator.go` to `256 * 1024`.
- [x] 5. Update mock/validation tests in `token/services/tokens/validation_test.go` to match the new `256 * 1024` limit behavior where applicable.
- [x] 6. Verify formatting, run code checks, and update plan progress.

## Notes & Decisions
- Setting the limit to `256 * 1024` (256KB) ensures all standard Idemix credentials, public keys, and zero-knowledge proof audit structures easily fit within the bounds while still offering robust protection against oversized payload resource exhaustion attacks (well below the `2 * 1024 * 1024` overall payload limit).

✅ COMPLETE
