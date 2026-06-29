# Wallet ID Validation for Pending Transactions

## Overview

The Fabric Token SDK implements wallet ID validation to ensure that all transactions written to the database reference valid, registered wallets. This validation is a critical security measure that prevents malformed records from corrupting vault state, audit trails, and balance calculations.

## Motivation

Before committing any write, the store service must validate that the payload satisfies the structural and relational invariants of the target table. Specifically:

- A `Transaction` row must reference enrollment IDs (wallet IDs) that correspond to registered wallets
- Invalid wallet IDs could lead to:
  - Corrupted vault state
  - Incorrect audit trails
  - Failed balance calculations
  - Orphaned transaction records

## Implementation

### Architecture

The validation is implemented in the `ttxdb.StoreService` and follows these principles:

1. **Fail Fast**: Validation occurs BEFORE the database transaction begins
2. **Cheap Checks**: Only validates wallet existence, not expensive consistency sweeps
3. **Optional**: Backward compatible - validation is skipped if wallet service not configured
4. **Comprehensive**: Validates all enrollment IDs (both senders and recipients)

### Code Location

- **Implementation**: `token/services/storage/ttxdb/store.go`
- **Tests**: `token/services/storage/ttxdb/wallet_validation_test.go`

### Key Components

#### WalletService Interface

```go
type WalletService interface {
    // OwnerWalletIDs returns the list of registered owner wallet identifiers
    OwnerWalletIDs(ctx context.Context) ([]string, error)
}
```

This interface abstracts the wallet registry, allowing the store service to validate wallet IDs without tight coupling to the identity service.

#### Validation Method

The `validateWalletIDs` method:

1. Extracts unique enrollment IDs from transaction records
2. Retrieves registered wallet IDs from the wallet service
3. Validates each enrollment ID against the registry
4. Returns a descriptive error listing all invalid IDs

#### Integration Point

Validation is called in `AppendTransactionRecord` before writing to the database:

```go
func (d *StoreService) AppendTransactionRecord(ctx context.Context, req *token.Request) error {
    // ... parse transaction records ...
    
    // Validate wallet IDs before writing to database
    if err := d.validateWalletIDs(ctx, txs); err != nil {
        return errors.WithMessagef(err, "wallet validation failed for transaction [%s]", req.Anchor)
    }
    
    // ... write to database ...
}
```

## Usage

### Setting Up Validation

To enable wallet validation, inject a wallet service into the store service:

```go
storeService, err := ttxdb.GetByTMSId(sp, tmsID)
if err != nil {
    return err
}

// Get wallet service from identity provider
walletService, err := identity.GetWalletService(sp, tmsID)
if err != nil {
    return err
}

// Enable validation
storeService.SetWalletService(walletService)
```

### Backward Compatibility

If the wallet service is not set, validation is automatically skipped:

```go
storeService, err := ttxdb.GetByTMSId(sp, tmsID)
// Validation will be skipped - backward compatible
```

## Validation Rules

### Valid Scenarios

1. **All wallet IDs registered**: Transaction proceeds normally
2. **Empty enrollment IDs**: Allowed for operations like:
   - Issue (no sender)
   - Redeem (no recipient)
3. **No wallet service configured**: Validation skipped (backward compatible)

### Invalid Scenarios

1. **Unregistered sender**: Transaction rejected with error listing invalid wallet ID
2. **Unregistered recipient**: Transaction rejected with error listing invalid wallet ID
3. **Multiple invalid IDs**: All invalid IDs listed in error message
4. **Wallet service error**: Transaction rejected, error propagated

## Error Handling

When validation fails, a descriptive error is returned:

```
wallet validation failed for transaction [tx123]: invalid wallet IDs (enrollment IDs not registered): [alice, bob]
```

This error:
- Identifies the transaction ID
- Lists all invalid wallet IDs
- Prevents the transaction from being written to the database

## Performance Considerations

### Design Decisions

1. **O(1) Lookup**: Uses a map for wallet ID lookups
2. **Single Query**: Retrieves all registered wallets once per validation
3. **Deduplication**: Validates each unique enrollment ID only once
4. **Early Exit**: Fails immediately if wallet service returns error

### Scale Considerations

- **Small to Medium Scale**: Validation adds negligible overhead
- **Large Scale**: Consider:
  - Caching registered wallet IDs
  - Batch validation for multiple transactions
  - Background reconciliation for expensive consistency checks

## Testing

Comprehensive test coverage includes:

1. **Valid wallets**: All IDs registered
2. **Invalid sender**: Sender not registered
3. **Invalid recipient**: Recipient not registered
4. **Multiple invalid**: Multiple IDs not registered
5. **Empty IDs**: Issue/redeem operations
6. **No wallet service**: Backward compatibility
7. **Wallet service errors**: Error propagation
8. **Duplicate IDs**: Efficient deduplication

Run tests:

```bash
go test ./token/services/storage/ttxdb -run TestValidateWalletIDs -v
```

## Security Benefits

1. **Data Integrity**: Prevents malformed records in the database
2. **Audit Trail**: Ensures all transactions reference valid participants
3. **Balance Accuracy**: Prevents orphaned tokens affecting balance calculations
4. **Early Detection**: Catches configuration errors before they propagate
5. **Fail Safe**: Rejects invalid transactions rather than accepting them

## Future Enhancements

Potential improvements:

1. **Caching**: Cache registered wallet IDs to reduce lookups
2. **Batch Validation**: Validate multiple transactions in a single call
3. **Async Validation**: Background validation for non-critical paths
4. **Metrics**: Track validation failures for monitoring
5. **Configurable**: Make validation optional via configuration

## Related Documentation

- [Storage Service](../services/storage.md)
- [Identity Service](../services/identity.md)
- [Transaction Flow](../services/ttx.md)