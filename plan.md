# Plan: Clean Unused Pseudonyms When Reducing Cache Size (#1820)

## Goal Description
When the pseudonym cache size is reduced and the service restarts, previously pre-generated cached pseudonyms remain on disk. To prevent this, pre-generated cached pseudonyms should not be persisted on disk (they should be ephemeral). We implement an `EphemeralIdentity()` method that uses `Temporary: true` in the nym key derivation options, so that cached pseudonyms do not persist in the KeyStore.

## Implementation Progress
- [x] Add `identityWithOpts()` internal method and `EphemeralIdentity()` to `idemix.KeyManager` (km.go)
- [x] Add `EphemeralIdentity()` to `idemixnym.KeyManager` (km.go)
- [x] Update `idemix.kmp.go` to wire cache backend to use `EphemeralIdentity`
- [x] Add unit tests for `EphemeralIdentity` in `km_test.go`
- [x] Add unit test for cache using ephemeral identities in `kmp_test.go`
- [x] Run unit tests and verify
- [x] Fix fp256bn_amcl references in TestEphemeralIdentity in km_test.go
- [x] Fix fp256bn_amcl references in TestCacheUsesEphemeralIdentity in kmp_test.go
- [x] Remove/fix FP256BN_AMCL benchmark in km_bench_test.go
- [x] Verify that all unit tests pass successfully
- [x] Create walkthrough summary

## Status: ✅ COMPLETE

## Notes & Decisions
- Checked `membership.KeyManager` interface. It doesn't have `EphemeralIdentity`.
- To avoid modifying `membership.KeyManager` and causing ripple effects, we use type assertion on `keyManager` in `kmp.go` to check for `EphemeralIdentity` method.
- The `fp256bn_amcl/idemix` testdata was deleted in main branch commit `a10d0d27` and replaced by `bls12_381_bbs_gurvy/idemix`. We must update the new tests/benchmarks in this branch to use the active curves.
