# Token Transaction (TTX) Service

The **Token Transaction (TTX) Service** is the primary orchestration component of the Fabric Token SDK. it provides a high-level API and a set of [Fabric Smart Client (FSC)](https://github.com/hyperledger-labs/fabric-smart-client) Views to help developers assemble, sign, and commit token transactions in a backend-agnostic manner.

The TTX service is designed with a **dependency injection pattern** (located in `token/services/ttx/dep`), which decouples the transaction orchestration logic from the underlying infrastructure providers like the Network Service, TMS Provider, and Storage Service.

## Transaction Lifecycle

The lifecycle of a token transaction typically involves the following stages, coordinated by the TTX service:

```mermaid
sequenceDiagram
    autonumber
    participant Initiator
    participant Recipient
    participant Auditor
    participant Network as Network Service
    participant Ledger as DLT / Ledger

    box darkgreen Token SDK Stack
        participant Initiator
        participant Recipient
        participant Auditor
        participant Network
    end

    Note over Initiator: 1. Request Identities
    Initiator->>+Recipient: RequestRecipientIdentityView
    Recipient-->>-Initiator: Recipient Data (Identity + Audit Info)

    Note over Initiator: 2. Assemble Request
    Initiator->>+Initiator: Issue / Transfer / Redeem operations

    Note over Initiator: 3. Collect Endorsements
    Initiator->>+Initiator: Sign locally
    Initiator->>+Recipient: Request Signatures (for spent tokens)
    Recipient-->>-Initiator: Signature
    Initiator->>+Auditor: AuditApproveView
    Auditor-->>-Initiator: Auditor Signature
    Initiator->>+Network: RequestApproval (DLT Endorsement)
    Network-->>-Initiator: Endorsed Envelope

    Note over Initiator: 4. Distribution & Ordering
    Initiator->>+Recipient: Distribute Transaction Metadata
    Initiator->>+Network: Broadcast Transaction
    Network->>+Ledger: Submit to Orderer

    Note over Initiator: 5. Finality Tracking
    Initiator->>+Network: Listen for Finality
    Ledger-->>-Network: Transaction Committed
    Network-->>-Initiator: Notify Finality
```

## Transaction Creation

Transactions are instantiated via the `ttx.NewTransaction` function. A transaction is anchored to a specific **Token Management System (TMS) ID**, which defines the network, channel, and namespace for the transaction.

When a transaction is created, it:
*   Initializes a new `token.Request` via the TMS.
*   Assigns a unique Transaction ID.
*   Registers a cleanup hook in the FSC view context to ensure resources (like locked tokens) are released if the transaction fails.

## Identity Management

To issue or transfer tokens, the initiator must acquire the recipient's identity. The TTX service provides interactive protocols for this purpose.

### Requesting Recipient Identities
The `RequestRecipientIdentityView` allows an initiator to request a fresh recipient identity from a counterparty. This process ensures that:
*   The recipient generates a fresh pseudonym (e.g., an Idemix pseudonym) to preserve privacy.
*   The initiator receives the necessary audit information to verify the identity's validity against the TMS public parameters.
*   Both parties update their local `IdentityInfo` stores to map the new identity to the correct FSC node.

### Recipient Protocol Flows (`recipients.go`)

The recipient identity protocols are implemented in `token/services/ttx/recipients.go`. The diagrams below document the on-wire messages exchanged by the initiator and responder views.
`RecipientData` can carry `Identity`, `AuditInfo`, `TokenMetadata`, and `TokenMetadataAuditInfo`.

Wire messages use JSON sessions (`token/services/utils/json/session`); the diagrams name the Go types being sent or received.

**Response paths (today).** In `RespondRequestRecipientIdentityView`, after the wallet lookup:

- If `recipientRequest.RecipientData != nil`, the responder checks `OwnerWallet.Contains` for `RecipientData.Identity`, then sends **the same** `RecipientData` value back on the session (echo path). The responder does **not** substitute wallet-held `AuditInfo` / metadata into that payload; what goes on the wire is the initiator-supplied structure (after the contains check).
- If `recipientRequest.RecipientData == nil`, the responder calls `OwnerWallet.GetRecipientData` and sends that **wallet-produced** `RecipientData` (fresh path).

In both cases the initiator receives a full `RecipientData` over the wire. The initiator then calls `WalletManager.RegisterRecipientIdentity` with that payload and updates the endpoint resolver. **Protocol hardening** may replace full-object responses with a minimal acknowledgement (or another minimal wire type) so the responder does not ship an entire `RecipientData` when a slimmer response suffices; that is a separate code change from this documentation.

**Multisig.** When `RecipientRequest.MultiSig` is true, the initiator may send an additional `MultisigRecipientData` after the first exchange; the responder registers identities and updates bindings as in code.

#### `RequestRecipientIdentityView` / `RespondRequestRecipientIdentityView`

```mermaid
sequenceDiagram
    autonumber
    participant I as Initiator
    participant R as Responder

    rect rgba(230, 230, 250, 0.35)
        Note over I,R: Phase 1 - Identity request
        I->>R: RecipientRequest{TMSID, WalletID, RecipientData?, MultiSig}
    end

    rect rgba(255, 245, 238, 0.5)
        Note over R: Phase 2 - Responder decision
        alt RecipientRequest.RecipientData != nil
            R->>R: Verify wallet contains RecipientData.Identity
            R-->>I: RecipientData (echo path)
        else RecipientRequest.RecipientData == nil
            R->>R: Generate RecipientData from wallet
            R-->>I: RecipientData (fresh data path)
        end
    end

    rect rgba(240, 255, 240, 0.45)
        Note over I,R: Phase 3 - Local registration and bindings
        I->>I: RegisterRecipientIdentity(RecipientData)
        I->>I: endpoint.Bind(requested FSC identity, RecipientData.Identity)
        R->>R: endpoint.Bind(context.Me, RecipientData.Identity)
    end

    rect rgba(245, 245, 245, 0.55)
        Note over I,R: Optional multisig extension
        opt MultiSig == true
            I-->>R: MultisigRecipientData{RecipientData, Nodes, Recipients}
            R->>R: RegisterRecipientIdentity(multisig data)
            R->>R: endpoint.Bind(each Node -> Recipient)
        end
    end

    Note over I,R: Full RecipientData on wire today (echo of request vs wallet-generated)
```

#### `ExchangeRecipientIdentitiesView` / `RespondExchangeRecipientIdentitiesView`

```mermaid
sequenceDiagram
    autonumber
    participant I as Initiator
    participant R as Responder

    rect rgba(230, 230, 250, 0.35)
        Note over I,R: Phase 1 - Exchange request
        I->>R: ExchangeRecipientRequest{TMSID, WalletID, RecipientData(local)}
    end

    rect rgba(255, 245, 238, 0.5)
        Note over R: Phase 2 - Responder processing
        R->>R: RegisterRecipientIdentity(request.RecipientData)
        R->>R: GetRecipientData(responder wallet)
        R-->>I: RecipientData(responder)
    end

    rect rgba(240, 255, 240, 0.45)
        Note over I,R: Phase 3 - Local registration and bindings
        I->>I: RegisterRecipientIdentity(remote RecipientData)
        I->>I: endpoint.Bind(other FSC identity, remote RecipientData.Identity)
        R->>R: endpoint.Bind(session caller, request.RecipientData.Identity)
    end

    Note over I,R: Full RecipientData on wire today (responder sends local wallet RecipientData)
```

### PolicyIdentity — Boolean-Expression-Governed Ownership

The TTX service supports **PolicyIdentity** owners: tokens whose spending requires satisfying a boolean expression over a set of component identities.  This enables richer access-control than simple multisig (M-of-N) — for example, an OR clause where any single co-owner may spend unilaterally, or complex nested expressions.

#### Creating a PolicyIdentity

Call `RequestPolicyIdentity` (in `token/services/ttx/recipients.go`) to negotiate a composite identity from all co-owners before building the transfer:

```go
recipient, err := bptx.RequestRecipientIdentity(ctx, "$0 OR $1",
    []view.Identity{bobFSCIdentity, charlieFSCIdentity},
    token.WithTMSIDPointer(tmsID),
)
```

Each co-owner's node responds with its component identity; the SDK assembles the `PolicyIdentity` envelope automatically.

#### Policy Expression Syntax

| Expression | Meaning |
|:-----------|:--------|
| `$0 OR $1` | Either component 0 **or** component 1 can spend alone. |
| `$0 AND $1` | Both component 0 **and** component 1 must sign. |
| `($0 OR $1) AND $2` | One of the first two parties plus party 2 must sign. |

`$N` is a zero-based index into the ordered component identity list supplied when creating the token.

#### Spending — OR Policy

For an OR policy the initiator alone can satisfy the policy.  Pass `WithPolicySigners` to restrict signature collection to only the signing party's slot; the remaining slots are left nil (which is valid for OR branches):

```go
_, err = context.RunView(ttx.NewCollectEndorsementsView(tx,
    ttx.WithPolicySigners(myComponentIdentity),
))
```

#### Spending — AND Policy

For an AND policy all co-owners must endorse.  Use `RequestSpendView` (in `token/services/ttx/boolpolicy/spend.go`) to notify co-owners before assembling the transaction, then collect endorsements from all components without restriction:

```go
_, err = context.RunView(bptx.NewRequestSpendView(unspentToken, serviceOpts...))
// ... build tx ...
_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
```

Co-owners run `EndorseSpendView` (via `EndorseSpend`) on their side, which ACKs the spend request and then endorses the assembled transaction.

#### Wallet and Authorization

The `boolpolicy.OwnerWallet` (in `token/services/ttx/boolpolicy/wallet.go`) wraps a standard owner wallet and filters the token list to policy-type tokens.  `VerifyApprover` can be used to assert that a given identity is one of the named component identities before allowing a spend.

The `EscrowAuth` struct (in `token/services/ttx/boolpolicy/auth.go`) implements the `Authorization` interface: `IsMine` returns true if any component identity of the policy token belongs to one of the node's owner wallets.

## Token Operations

The TTX service supports three primary operations through the `TokenRequest` API:

### Issue
Allows authorized issuers to create new tokens. The service:
1.  Retrieves the issuer's identity from the internal **Identity Service**.
2.  Generates an "Issue Action" using the driver-specific logic.
3.  Adds the action and its metadata to the transaction request.

### Transfer
Enables the transfer of token ownership. The service:
1.  Uses the **Selector Service** to pick spendable tokens (UTXOs) that cover the requested amount.
2.  Locks the selected tokens in the local `TokenLocks` table to prevent double-spending.
3.  Generates a "Transfer Action" that consumes the selected tokens and creates new ones for the recipients.

### Redeem
A specialized transfer where the recipient is "hidden" or "empty," effectively removing the tokens from circulation on the ledger.

Redeem supports an enhanced flow where an issuer signature is required as part of transfer validation:
1. Add the redeem action with `tx.Redeem(...)`.
2. If the issuer endpoint cannot be resolved automatically, pass `ttx.WithFSCIssuerIdentity(...)` so the initiator can contact the issuer for endorsement.
3. Optionally pass `ttx.WithIssuerPublicParamsPublicKey(...)` to pin which issuer public-parameters signing key must authorize the redeem.
4. Run `CollectEndorsementsView` to collect owner, auditor (if configured), and issuer signatures.

## Collecting Endorsements

The `CollectEndorsementsView` is responsible for gathering all signatures required to make a transaction valid:
*   **Owner Signatures**: For every token spent, the service requests a signature from the node that owns the corresponding identity.
*   **Issuer Signatures**: For transactions involving token issuance and enhanced redeem flows that require issuer authorization.
*   **Auditor Signatures**: If the TMS is configured with an auditor, the transaction must be approved via the `AuditApproveView`.
*   **Network Endorsements**: The service delegates to the **Network Service** to obtain backend-specific endorsements (e.g., Fabric chaincode endorsements).

## Distribution and Ordering

Once fully endorsed, the transaction metadata is distributed to all recipients so they can track and spend their new tokens. The initiator then uses the `OrderingView` to broadcast the transaction envelope to the network's ordering service.

## Finality and Discovery

The `FinalityView` allows applications to wait for a transaction to be committed to the ledger. Internally, the SDK's **Network Service** listens for ledger events. When a transaction reaches finality, the Network Service notifies the SDK, which then updates the local `Transactions DB` and `Tokens DB` to reflect the new state.

### Transaction Recovery

The SDK includes an automatic recovery mechanism to handle pending transactions that may have lost their finality listeners due to node restarts, network interruptions, or other failures. 
The recovery service is part of the **Storage Service** and is instantiated by the **Network Service** to recover transactions from either `TTXDB` (for regular transactions) or `AuditDB` (for auditor nodes).

For detailed information about the recovery mechanism, see:
- [Storage Service - Transaction Recovery](storage.md#transaction-recovery-service)
- [Configuration Guide - Recovery Parameters](../configuration.md), Section `Optional: token.tms.<name>.services.network.fabric.recovery`
