# Database Operation Timeouts

## Overview

To prevent resource exhaustion attacks and ensure system reliability, all database operations in the Fabric Token SDK now have explicit timeouts. This prevents slow, deadlocked, or malicious database operations from holding resources indefinitely.

## Security Rationale

### The Problem

Without timeouts on database operations:
- **Resource Exhaustion**: Slow DB queries can block goroutines indefinitely, leading to memory exhaustion
- **Connection Pool Starvation**: Blocked operations hold database connections, preventing new operations
- **Lock Contention**: Database locks held indefinitely can cascade into system-wide deadlocks
- **DoS Vulnerability**: Attackers can trigger expensive queries to exhaust system resources

### The Solution

Every database operation now uses a context with an explicit timeout:
- **Prevents indefinite blocking**: Operations fail fast when DB is unresponsive
- **Ensures resource cleanup**: Context cancellation releases connections and locks
- **Enables graceful degradation**: System can detect and respond to DB performance issues
- **Provides DoS protection**: Limits impact of expensive or malicious queries

## Timeout Configuration

### Default Timeouts

Three timeout tiers are provided based on operation complexity:

| Timeout Type | Duration | Use Case |
|--------------|----------|----------|
| **Short** | 5 seconds | Simple operations (locks, inserts, deletes) |
| **Medium** | 15 seconds | Standard queries and updates |
| **Long** | 30 seconds | Batch operations and complex queries |

### Configuration

Timeouts are configured via `DBTimeoutConfig`:

```go
config := &common.DBTimeoutConfig{
    ShortOpTimeout:  5 * time.Second,
    MediumOpTimeout: 15 * time.Second,
    LongOpTimeout:   30 * time.Second,
}
```

## Implementation

### Context Wrapper Functions

Four helper functions wrap contexts with appropriate timeouts:

```go
// Short timeout for quick operations
timeoutCtx, cancel := common.WithShortTimeout(ctx, nil)
defer cancel()

// Medium timeout for standard operations
timeoutCtx, cancel := common.WithMediumTimeout(ctx, nil)
defer cancel()

// Long timeout for batch operations
timeoutCtx, cancel := common.WithLongTimeout(ctx, nil)
defer cancel()

// Custom timeout for specific needs
timeoutCtx, cancel := common.WithCustomTimeout(ctx, 42*time.Second)
defer cancel()
```

### Resource Cleanup

**Critical**: Always use `defer cancel()` immediately after creating a timeout context:

```go
timeoutCtx, cancel := common.WithShortTimeout(ctx, nil)
defer cancel() // REQUIRED: Releases resources even if operation succeeds

_, err := db.ExecContext(timeoutCtx, query, args...)
```

This ensures:
- Timers are stopped when operation completes
- Resources are released on timeout
- No goroutine leaks occur

## Modified Operations

### Token Lock Operations

**File**: `token/services/storage/db/sql/common/tokenlock.go`

```go
func (db *TokenLockStore) Lock(ctx context.Context, tokenID *token.ID, consumerTxID transaction.ID) error {
    timeoutCtx, cancel := WithShortTimeout(ctx, nil)
    defer cancel()
    
    // ... query construction ...
    _, err := db.WriteDB.ExecContext(timeoutCtx, query, args...)
    return err
}
```

**Operations**:
- `Lock()` - Token locking with 5s timeout
- `UnlockByTxID()` - Token unlocking with 5s timeout

### Transaction Storage

**File**: `token/services/storage/db/sql/common/transactions.go`

```go
func (db *TransactionStore) SetStatus(ctx context.Context, txID string, status dbdriver.TxStatus, message string) error {
    timeoutCtx, cancel := WithShortTimeout(ctx, nil)
    defer cancel()
    
    // ... query construction ...
    _, err = db.writeDB.ExecContext(timeoutCtx, query, args...)
    return err
}
```

**Operations**:
- `AddTransactionEndorsementAck()` - 5s timeout
- `SetStatus()` - 5s timeout
- `AddTransactionRecord()` - 15s timeout (batch operation)
- `AddTokenRequest()` - 15s timeout
- `AddMovement()` - 15s timeout (batch operation)

### Token Operations

**File**: `token/services/storage/db/sql/common/tokens.go`

```go
func (db *TokenStore) DeleteTokens(ctx context.Context, ids ...*token.ID) error {
    timeoutCtx, cancel := WithShortTimeout(ctx, nil)
    defer cancel()
    
    // ... query construction ...
    _, err := db.writeDB.ExecContext(timeoutCtx, query, args...)
    return err
}
```

**Operations**:
- `DeleteTokens()` - 5s timeout
- `TokenTransaction.Delete()` - 5s timeout
- `TokenTransaction.StoreToken()` - 5s timeout

## Error Handling

### Timeout Errors

When a timeout occurs, the operation returns a context deadline exceeded error:

```go
_, err := db.ExecContext(timeoutCtx, query, args...)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // Handle timeout specifically
        logger.Warnf("Database operation timed out: %v", err)
        return errors.Wrap(err, "database operation timeout")
    }
    // Handle other errors
    return err
}
```

### Best Practices

1. **Log timeout occurrences**: Track patterns that may indicate DB performance issues
2. **Monitor timeout rates**: High timeout rates suggest infrastructure problems
3. **Adjust timeouts if needed**: Some operations may legitimately need longer timeouts
4. **Investigate root causes**: Timeouts are symptoms, not solutions

## Testing

### Unit Tests

**File**: `token/services/storage/db/sql/common/timeout_test.go`

Tests verify:
- Default timeout values are correct
- Context deadlines are set properly
- Cancellation works correctly
- Custom configurations are respected
- Context values are inherited

### Integration Tests

Integration tests should verify:
- Operations complete within timeout under normal conditions
- Operations fail gracefully when DB is slow
- Resources are released on timeout
- No goroutine leaks occur

## Migration Guide

### For Existing Code

If you have custom database operations, add timeouts:

**Before**:
```go
_, err := db.ExecContext(ctx, query, args...)
```

**After**:
```go
timeoutCtx, cancel := common.WithShortTimeout(ctx, nil)
defer cancel()
_, err := db.ExecContext(timeoutCtx, query, args...)
```

### Choosing the Right Timeout

- **Short (5s)**: Single-row inserts, updates, deletes, simple locks
- **Medium (15s)**: Multi-row queries, joins, aggregations
- **Long (30s)**: Batch operations, complex queries, migrations
- **Custom**: Operations with known specific requirements

## Performance Impact

### Overhead

Minimal overhead from timeout implementation:
- Context creation: ~100ns
- Timer setup: ~1µs
- Cancellation: ~100ns

### Benefits

Significant benefits under load:
- Prevents resource exhaustion
- Enables faster failure detection
- Improves system predictability
- Protects against DoS attacks

## Future Enhancements

Potential improvements:
1. **Configurable timeouts**: Load from configuration files
2. **Adaptive timeouts**: Adjust based on observed DB performance
3. **Circuit breakers**: Fail fast when DB is consistently slow
4. **Metrics integration**: Track timeout rates and patterns
5. **Retry logic**: Automatic retry with backoff for transient failures

## References

- [Context Package Documentation](https://pkg.go.dev/context)
- [Hyperledger Security Policy](../../SECURITY.md)
- [Database Best Practices](../development/storage.md)