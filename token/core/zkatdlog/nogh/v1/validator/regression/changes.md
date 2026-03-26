# Report on Changes

This file describe the changes that required the regeneration of the regression test data.

## With respect to commit `73af27c1cb3f7b83a49f8bf247fc88ea219fc374`

We replace `bytes.Join` with a more memory efficient version `crypto.AppendFixed32` (`token/core/common/crypto/slice.go`).
The test under `token/core/common/crypto/slice_alloc_test.go` checks that this is indeed the case.
## With respect to commit `refactor/identity-type-int32`
Changed `IdentityType` from `string` to `int32` in `token/driver/wallet.go`.
This changes the ASN1 serialization of `TypedIdentity` from UTF8String (tag 19) to INTEGER (tag 2),
requiring regeneration of all regression testdata that contains serialized typed identities
(auditor keys, owner identities).
