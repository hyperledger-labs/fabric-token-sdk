# Interoperability via Scripting

Token SDK supports interoperability, such as HTLC mechanism (see related section), via scripting. 
It allows spending a token to a script by encoding the script in the `Owner` field of a `Token`, and the different drivers are capable of interpreting the owner as a script.

## HTLC (exchange)

HTLC (Hash Time Locked Contract) is a token transfer that use hashlocks and timelocks to require that the recipient of a token either acknowledge receiving the token prior to a deadline by generating cryptographic proof or forfeit the ability to claim the token, returning it to the sender.

### HTLC script

The HTLC (exchange) script encodes the details of the exchange, the identities of the sender and the recipient of the token, a deadline, and hashing information.

```go
// Script contains the details of an exchange
type Script struct {
    Sender    view.Identity
    Recipient view.Identity
    Deadline  time.Time
    HashInfo  HashInfo
}

// HashInfo contains the information regarding the hashing
type HashInfo struct {
    Hash         []byte
    HashFunc     crypto.Hash
    HashEncoding encoding.Encoding
}
```

## Interoperability services

Some interoperability services, located in `token/services/interop`, which are responsible for assembling the token transaction, managing the transaction lifecycle, and so on, are the same as the `Token Transaction Services`. 
Other services are script specific. 

For example, the token transaction assembling service enables appending exchange, claim, or reclaim actions to the token request of the transaction. All of these actions translate into a transfer action. 
The interop `Wallet` service supports listing expired tokens, whose deadline have passed, and listing tokens with a desired matching preimage.
In addition, the interop signer and verifier services are script specific, for example in the HTLC case the preimage is part of the signed message.

## Driver adjustments 

The `FabToken` and `ZKAT DLog` drivers support also interoperability, and more specifically, the drivers support HTLC.

In addition to the regular validation process, their `Validator` ensures that in the exchange and claim cases the deadline has not expired, and in the reclaim case the deadline has passed.    
Moreover, the validator returns a `TransferAction` now holds the `ClaimPreImage`, as it is written into the ledger and can later be searched by the scanner service located in `token/services/interop/scanner.go`

The `deserializer` in the interoperability case returns a specialized script owner verifier, that takes into account both the sender and the recipient as well as the deadline (located in `token/services/interop/signer.go`). 

The driver's `TransferService` also takes into account the presence of scripts, as `Transfer` returns `TransferMetadata` which includes information for both the sender and recipient of a script.

Lastly, the `auditor` inspects the token ownership also in the interoperability case, and verifies that the audit info matches the script owner's, both the sender and the recipient.
