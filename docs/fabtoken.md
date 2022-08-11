# FabToken

The `FabToken` driver is a simple implementation of the Driver API that does not support privacy.
Therefore, the ledger reveals to any third party who did what. 

## Public Params Manager

`FabToken` understands the following public parameters:

```go
type PublicParams struct {
	// This indicates that this is the public parameters for fabtoken driver
	Label string
	// The precision of token quantities
	QuantityPrecision uint64
	// This is set when audit is enabled
	Auditor []byte
	// This encodes the list of authorized auditors
	Issuers [][]byte
}
```

The `Label` field must be set to `"fabtoken"`.
`FabToken` supports multiple issuers and a single auditor.

## IdentityProvider

In `FabToken`, the only long-term identities supported are `X509-based Fabric MSP identities`.
Such an identity contains an X509 certificate and reveal in the clear the Enrollment ID of the certificate's owner.

The `Auditor` and `Issuers` fields of the public parameters must contain serialized version of X509-based Fabric MSP identities.  

## Wallet Service

Each wallet is bound to a single `X509-based Fabric MSP identities`. 

## Token Service

A token is represented on the ledger as the `json` representation of the `token.Token` struct.
The `Owner` field of the `token.Token` struct is filled with the `asn1` representation of the `identity.RawOwner` struct
whose `Type` field is set to `identity.SerializedIdentityType` and whose `Raw` field is set to
a serialized X509-based Fabric MSP identity.

## Validator

`FabToken` validation process ensures the following:
- Only the issuers whose identity is registered in the public parameters (`Issuers` field) are allowed to issue tokens.
  If the public parameters do not contain any issuer (`Issuers` field is empty), then anyone is allowed to issue tokens.
- Only the rightful owners of the tokens are allowed to transfer them.
- In transfer operation, the sum of the inputs must be equal to the sum of the outputs.
- Only the owner of a token can redeem it.
- If the public parameters contain an auditor, then the auditor must sign the token request for it to be considered valid.