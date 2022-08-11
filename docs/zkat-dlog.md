# ZKAT DLog

The `ZKAT DLog` driver supports privacy using Zero Knowledge Proofs. 
We follow a simplified version of the blueprint described in the paper <!-- markdown-link-check-disable -->
[`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')<!-- markdown-link-check-disable -->
by Androulaki et al.

`ZKAT` stands for `Zero Knowledge Asset Transfer` and `DLog` stands for `Discrite Logarithm` to mean
that the scheme is secure under `computational assumptions in bilinear groups` in the `random-oracle model`.

In more details, the driver hides the token's owner, type, and quantity. But it reveals which token has been spent by
a give transaction. We say that this driver does not support `graph hiding`.
Owner anonymity and unlinkability is achieved by using Identity Mixer (Idemix, for short).

The identity of the issuers and the auditors is not hidden. 

## Public Params Manager

`ZKAT DLog` understands the following public parameters:

```go
type PublicParams struct {
	// Label is the identifier of the public parameters.
	Label string
	// Curve is the pairing-friendly elliptic curve used for everything but Idemix.
	Curve math.CurveID
	// PedGen is the generator of the Pedersen commitment group.
	PedGen *math.G1
	// PedParams contains the public parameters for the Pedersen commitment scheme.
	PedParams []*math.G1
	// RangeProofParams contains the public parameters for the range proof scheme.
	RangeProofParams *RangeProofParams
	// IdemixCurveID is the pairing-friendly curve used for the idemix scheme.
	IdemixCurveID math.CurveID
	// IdemixIssuerPK is the public key of the issuer of the idemix scheme.
	IdemixIssuerPK []byte
	// Auditor is the public key of the auditor
	Auditor []byte
	// Issuers is a list of public keys of the entities that can issue tokens.
	Issuers [][]byte
	// QuantityPrecision is the precision used to represent quantities
	QuantityPrecision uint64
	// Hash is the hash of the serialized public parameters.
	Hash []byte
}
```

The `Label` field must be set to `"zkatdlog"`.
`ZKAT DLog` supports multiple issuers and a single auditor.

## IdentityProvider

In `ZKAT DLog`, there are two  long-term identities supported: 
- `X509-based Fabric MSP identities`. Such an identity contains an X509 certificate and reveal in the clear the Enrollment ID of the certificate's owner.
  It will be used for issuers and auditors.
- "Idemix-based Fabric MSP Identities". Such an identity contains an idemix credential.
  It will be used for token owners.

## Wallet Service


## Token Service

A token is represented on the ledger as the `json` representation of the following data structure:

```go
// Token encodes Type, Value, Owner
type Token struct {
	// Owner is the owner of the token
	Owner []byte
	// Data is the Pedersen commitment to type and value
	Data *math.G1
}
```

Each token is associated with metadata stored in the following data structure:

```go
// TokenInformation contains the metadata of a token
type Metadata struct {
	// Type is the type of the token
	Type string
	// Value is the quantity of the token
	Value *math.Zr
	// BlindingFactor is the blinding factor used to commit type and value
	BlindingFactor *math.Zr
	// Owner is the owner of the token
	Owner []byte
	// Issuer is the issuer of the token, if defined
	Issuer []byte
}
```

## Issue Service

To be continued...

## Transfer Service

To be continued...

## Validator

To be continued...
