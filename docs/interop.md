# Interoperability via Scripting

Token SDK supports interoperability, cross-chain operations, via scripting. 
It allows spending a token to a script by encoding the script in the `Owner` field of a `Token`, and the different drivers are capable of interpreting the owner as a script.
After the ownership is assigned to a script, the script is evaluated at the time of spending the token. 
The right to spend the token is enforced according to the conditions within the script.

## HTLC

HTLC (Hash Time Locked Contract) is a token transfer that use hashlocks and timelocks to require that the recipient of a token either acknowledge receiving the token prior to a deadline by generating cryptographic proof or forfeit the ability to claim the token, returning it to the sender.
With this mechanism, the Token SDK supports atomic cross-chain swap of tokens. 

### HTLC script

The HTLC script encodes the details of the HTLC, the identities of the sender and the recipient of the token, a deadline, and hashing information.
The hashing information includes the hash itself, the hash function used (e.g., SHA-256), and the encoding (e.g., Base64).
The hash is chosen by the sender and the recipient must provide the preimage for the transfer to happen.

```go
// Script contains the details of an HTLC
type Script struct {
    Sender    view.Identity
    Recipient view.Identity
    Deadline  time.Time
    HashInfo  HashInfo
}

// HashInfo contains the information regarding the hash
type HashInfo struct {
    Hash         []byte
    HashFunc     crypto.Hash
    HashEncoding encoding.Encoding
}
```

## Interoperability services

The token transaction assembling service enables appending `Lock`, `Claim`, or `Reclaim` actions to the token request of the transaction. All of these actions translate into a transfer action. 
`Lock` is the locking process, where the sender sets the details of the HTLC and transfers ownership of the token to a script. 
`Claim` allows the recipient to gain ownership of the token by providing the preimage. 
`Reclaim` returns the token to the sender. 
Claim must happen before the deadline ends, while reclaim can only occur after the deadline has passed.

```go
func (t *Transaction) Lock(wallet *token.OwnerWallet, sender view.Identity, typ string, value uint64, recipient view.Identity, deadline time.Duration, opts ...token.TransferOption) ([]byte, error)
func (t *Transaction) Claim(wallet *token.OwnerWallet, tok *token2.UnspentToken, preImage []byte) error
func (t *Transaction) Reclaim(wallet *token.OwnerWallet, tok *token2.UnspentToken) error
```

The interop `Wallet` service, located under `token/services/interop/`, supports listing tokens with a desired matching preimage, and listing expired tokens, whose deadline have passed.

In addition, the interop `Signer` and `Verifier` services are script specific, for example in the HTLC case the preimage is part of the signed message.

Finally, the interoperability services which are responsible for assembling the token transaction and managing its lifecycle are the same as the [`Token Transaction Services`](./services.md).
They are located in `token/services/interop`.


## Driver adjustments 

The `FabToken` and `ZKAT DLog` drivers support also interoperability, and more specifically, the drivers support HTLC.

The validator in `FabToken` and `ZKAT DLog` can be enhanced with extra validators to accommodate additional validation rules. In particular, to support atomic swap, they take a validator that ensures HTLC conditions are met. That is, the deadline has not passed in the case of lock, that a claim was initiated by the recipient before the expiration of the deadline and carries the pre-image matching the hash, and that a reclaim is initiated by the sender after the deadline has passed.

Their `TransferAction` carries the pre-image at time of transaction assembly to support HTLC.

The `deserializer` in the interoperability case returns a specialized script owner verifier, that takes into account both the sender and the recipient as well as the deadline and the hash. 

The driver's `TransferService` also takes into account the presence of scripts, as `Transfer` returns `TransferMetadata` which includes information for both the sender and recipient of a script.

Lastly, the `auditor` inspects the token ownership also in the interoperability case, and verifies that the audit info matches the script owner's, both the sender and the recipient.

For more details on the drivers see [`FabToken`](./fabtoken.md) and [`ZKAT DLog`](./zkat-dlog.md).
