# Upgradability in Fabric Token SDK

This document provides a comprehensive guide to upgrading components within a Fabric Token SDK (FTS) application. Upgradability is essential for long-term maintenance, allowing for security patches, feature additions, and protocol migrations.

The SDK manages upgradability at three distinct layers:
1.  **Ledger Layer**: Upgrading existing tokens to new formats.
2.  **Driver Layer**: Managing compatibility between the SDK and underlying token implementations.
3.  **Storage Layer**: Handling local database schema evolutions.

---

## Token Upgradability (Ledger)

The SDK manages token upgrades using two distinct mechanisms: the **Atomic "Burn and Re-issue" protocol** for across-format migrations and **In-place Upgrades** for backward-compatible transitions.

### In-Place Upgrades

In-place upgrades allow the SDK to spend tokens from a previous driver version or format (e.g., Fabtoken) directly as if they were native to the current driver (e.g., ZKAT-DLOG), without requiring an explicit ledger transaction first.

#### Criteria for In-Place Upgrades:
The current driver determines compatibility based on several criteria:
1.  **Format Support**: The token's format must be included in the driver's `SupportedTokenFormats()`.
2.  **Precision Compatibility**: For Fabtoken to DLog upgrades, the original token's precision must be less than or equal to the current driver's maximum supported precision (e.g., 64-bit).
3.  **Automatic Commitment**: When the driver encounters a compatible legacy token (like Fabtoken), it automatically generates a Pedersen commitment and an **Upgrade Witness**. This witness allows the new driver to prove the validity of the original token while treating it as a zero-knowledge commitment in the new transaction.

### The "Burn and Re-issue" Mechanism

When in-place upgrade is not possible (e.g., moving to a completely incompatible cryptographic curve or increasing precision beyond limits), the SDK implements an atomic "Burn and Re-issue" protocol. 

#### Step-by-Step Flow:
1.  **Identification**: The owner identifies tokens that are no longer supported.
2.  **Challenge-Response**: The owner requests a "challenge" from an authorized issuer.
3.  **Proof Generation**: The owner generates an "upgrade proof" showing they own the old tokens and that the values match the intended new tokens.
4.  **Atomic Transaction**: The issuer verifies the proof and submits a transaction that consumes the old tokens and issues new ones.

### Code Example: Identifying Unsupported Tokens

Developers can use the `tokens` service to find tokens that require an upgrade.

```go
// Get the tokens service for a specific TMS
tms, _ := token.GetManagementService(context, token.WithTMSID(myTMSID))
tokensService, _ := tokens.GetService(context, tms.ID())

// Iterate over tokens of type "USD" that the current driver cannot spend
it, err := tokensService.UnsupportedTokensIteratorBy(
    context.Context(), 
    myWalletID, 
    "USD",
)
if err != nil {
    return err
}
defer it.Close()

var toUpgrade []token.LedgerToken
for {
    tok, _ := it.Next()
    if tok == nil { break }
    toUpgrade = append(toUpgrade, *tok)
}
```

### Code Example: The Upgrade Transaction (Issuer Side)

The issuer uses the `ttx` package to wrap the upgrade logic.

```go
// Inside a Responder View
tx, err := ttx.NewTransaction(context, nil, ttx.WithTMSID(upgradeRequest.TMSID))

// The Upgrade call consumes old tokens and issues new ones in one atomic step
err = tx.Upgrade(
    issuerWallet,
    upgradeRequest.RecipientIdentity,
    upgradeRequest.Challenge,
    upgradeRequest.Tokens, // Old tokens from ledger
    upgradeRequest.Proof,  // ZK-Proof or Signature
)
```

### Recommendations for Token Upgrades
*   **Batching**: Upgrade tokens in batches (e.g., 10-20 at a time). Large upgrades can exceed the maximum transaction or block size limits of the underlying ledger (e.g., Fabric's 10MB limit).
*   **Offline Owners**: Owners must be online to initiate an upgrade. Consider providing a UI notification when "Unspendable" tokens are detected.
*   **Verification**: Always verify the `PublicParameters` of the new driver before initiating a mass upgrade to ensure the target format is correct.

---

## Driver Upgradability

The SDK handles driver transitions gracefully during its startup sequence.

### Automatic Spendability Management

When the `Token Management Service (TMS)` initializes, it performs a `PostInit` sequence. It compares the formats of all tokens in the local database against the `SupportedTokenFormats()` reported by the currently loaded driver.

*   **Unsupported Formats**: Tokens with formats not recognized by the driver are marked `spendable = false` in the local DB.
*   **Supported Formats**: Tokens matching the current driver are marked `spendable = true`.

### How Drivers Define Formats

Drivers like `fabtoken` or `zkatdlog` derive their format string from their `PublicParameters` (e.g., precision, identity types).

```go
// Example of how a driver might calculate its format
func SupportedTokenFormat(precision uint64) (token.Format, error) {
    hasher := sha256.New()
    hasher.Write([]byte("zkatdlog"))
    hasher.Write([]byte(fmt.Sprintf("%d", precision)))
    return token.Format(hex.EncodeToString(hasher.Sum(nil))), nil
}
```

For more information on how token formats are used in the SDK's token service, see the [Tokens Service documentation](./services/tokens.md).

### Recommendations for Driver Upgrades
*   **Side-by-Side Migration**: If possible, deploy the new driver version on a subset of nodes first to verify transaction validation before a full network rollout.
*   **Monitor "Spendable" Balance**: Use the `Balance` API to monitor the ratio of spendable vs. unspendable tokens. A sudden drop in spendable balance indicates a driver mismatch.

---

## Storage DB Schema Upgradability

The local storage (SQL) uses a "Lazy Creation" strategy.

### Table Initialization logic

The SDK uses `CREATE TABLE IF NOT EXISTS`. This handles fresh installs perfectly but does not manage `ALTER TABLE` operations for existing databases.

```sql
-- The SDK executes this on startup
CREATE TABLE IF NOT EXISTS fsc_tokens (
    tx_id TEXT NOT NULL,
    idx INT NOT NULL,
    -- ... other columns
    ledger_type TEXT DEFAULT '', -- New columns added in SDK updates
    PRIMARY KEY (tx_id, idx)
);
```

### Handling Schema Changes

If a new version of the Token SDK adds a column (e.g., `ledger_metadata` or `spent_at`), the `IF NOT EXISTS` clause will prevent the new schema from being applied to an existing table.

### Recommendations for Schema Migrations
1.  **Manual SQL Scripts**: For production systems, maintain a set of SQL migration scripts. Before starting the updated SDK, run:
    ```sql
    ALTER TABLE fsc_tokens ADD COLUMN IF NOT EXISTS ledger_metadata BYTEA;
    ```
2.  **Vault Re-scan**: For non-critical nodes or during development, you can simply delete the local database file (e.g., `vault.db`). The SDK's `Vault` service can re-sync its state by scanning the ledger, though this may take time depending on the ledger size.
3.  **Check Release Notes**: Always check the SDK release notes for "Database Schema Changes" which will list any required manual `ALTER` statements.

---

## Serialization and Protocol Stability

The Token SDK relies heavily on **Protocol Buffers (Protobuf)** for serializing all core objects, including Public Parameters, Token Requests, and individual Actions. This choice is fundamental to the SDK's ability to evolve over time while maintaining compatibility between nodes running different software versions.

### The Role of Protobuf in Upgradability

Protobuf provides a binary serialization format that is both efficient and highly extensible. The SDK leverages several Protobuf features to ensure long-term stability:

1.  **Field Numbering and Compatibility**: 
    - **Backward Compatibility**: Newer versions of the SDK can add new fields to messages (e.g., adding an optional `Priority` field to a `TokenRequest`). Older nodes receiving these messages will simply ignore the unknown fields and continue processing the data they recognize.
    - **Forward Compatibility**: Newer nodes can receive messages from older nodes. Any missing fields in the older message are assigned their default values (e.g., `0` for integers, `""` for strings), allowing the new logic to handle them gracefully.

2.  **Opaque "Raw" Envelopes**: 
    The SDK uses a "wrapper" pattern for driver-specific data. For example, the `PublicParameters` message at the driver API level looks like this:
    ```protobuf
    message PublicParameters {
      string identifier = 1; // e.g., "zkatdlognogh/v1"
      bytes raw = 2;        // Opaque driver-specific bytes
    }
    ```
    This allows the core SDK to handle the delivery and storage of public parameters without needing to understand their internal structure. The `raw` bytes are only unmarshalled by the specific driver version identified by the `identifier`.

For more details on the specific Protobuf messages used by each driver, see:
- [**FabToken Protobuf Messages**](drivers/fabtoken.md#protobuf-messages)
- [**DLog (NOGH) Protobuf Messages**](drivers/dlogwogh.md#protobuf-messages)

3.  **Extensible Metadata**: 
    Most core messages (like `IssueMetadata` or `PublicParameters`) include a `map<string, bytes> extra_data` or `application` field. This allows developers to attach arbitrary information to transactions or configurations without modifying the underlying `.proto` definitions, avoiding the need for a full protocol migration for application-specific changes.

### Recommendations for Protocol Changes
*   **Never Reuse Field Numbers**: Once a field number is assigned in a `.proto` file, it must never be reassigned to a different field, even if the original field is deprecated.
*   **Prefer Optional Fields**: Use `proto3` defaults or explicitly check for presence to ensure that missing fields from older clients don't cause crashes.
*   **Versioned Packages**: For major, breaking changes in a driver's internal logic, create a new protobuf package (e.g., `package zkatdlognogh.v2;`). This allows both the old and new unmarshallers to coexist in the same codebase.

---

## Summary of Upgradability Responsibilities

| Component | Responsibility | Mechanism |
| :--- | :--- | :--- |
| **Tokens** | Owner / Issuer | `ttx.Transaction.Upgrade` (Burn & Re-issue) |
| **Driver** | Admin / SDK | `PostInit` (Automatic Spendability Toggle) |
| **Schema** | Developer / Admin | Manual SQL `ALTER` or Database Re-sync |
