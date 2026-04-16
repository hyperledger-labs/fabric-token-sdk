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

## Collecting Endorsements

The `CollectEndorsementsView` is responsible for gathering all signatures required to make a transaction valid:
*   **Owner Signatures**: For every token spent, the service requests a signature from the node that owns the corresponding identity.
*   **Issuer Signatures**: For transactions involving token issuance.
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
