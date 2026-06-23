# Keystore Cleanup Service

The **Keystore Cleanup Service** provides automatic deletion of cryptographic keys from the keystore for tokens that have been deleted (spent, expired, or invalidated). This ensures that the keystore doesn't accumulate stale keys indefinitely, improving security and reducing storage overhead.

## Overview

The cleanup system consists of three main components:

1. **Manager**: Orchestrates the cleanup process with periodic scanning and distributed coordination
2. **Identity Service**: Derives Subject Key Identifiers (SKIs) from owner identities for key deletion
3. **Storage**: Provides database operations for querying deleted tokens and tracking cleanup state

## Architecture

### Cleanup Manager

The Manager runs in the background and periodically scans for deleted tokens that are eligible for key cleanup. It uses distributed locking (PostgreSQL advisory locks) to ensure only one replica in a multi-instance deployment performs cleanup at a time.

**Key Features:**
- Periodic scanning with configurable intervals
- Worker pool for parallel key deletion
- Distributed leadership via advisory locks
- Idempotent operations with cleanup tracking

### Identity Service

The Identity Service derives SKIs from owner identities to identify which keys should be deleted from the keystore. It supports both Idemix and X.509 identity types.

**SKI Derivation:**
- Computes SHA256 hash of identity bytes as the SKI
- Works for both Idemix (audit info with public keys) and X.509 (certificates) identities
- Returns empty list for unsupported or empty identities

### Storage Interface

The Storage interface abstracts database operations needed for cleanup:
- `AcquireCleanupLeadership`: Obtains distributed lock for leader election
- `GetDeletedTokens`: Queries deleted tokens older than TTL that haven't been cleaned
- `MarkTokenCleaned`: Records successful key cleanup to prevent reprocessing

## Key Features

### Distributed Coordination
- Supports horizontal scaling with multiple replicas
- Leader election prevents conflicting cleanup attempts
- PostgreSQL advisory locks ensure only one instance performs cleanup

### Safety Guarantees
- TTL-based eligibility ensures tokens are truly finalized before cleanup
- Idempotent operations allow safe retries
- Partial failure handling continues processing other tokens

### Performance
- Configurable batch size and worker count
- Parallel key deletion via worker pool
- Efficient database queries with indexed columns

## Configuration

Cleanup behavior is controlled via configuration (see [Configuration](../configuration.md)):

```yaml
cleanup:
  enabled: false             # Disabled by default - must be explicitly enabled
  ttl: 24h                   # Minimum age before cleanup
  scanInterval: 1h           # How often to scan
  batchSize: 100            # Max tokens per scan
  workerCount: 4            # Parallel workers
  advisoryLockID: 0x74746b636c65616e  # Lock ID for leader election
  instanceID: "cleanup-1"   # Instance identifier (auto-generated if empty)
```

**Note:** The cleanup service is **disabled by default** and must be explicitly enabled in the configuration. This is a conservative default to prevent unexpected key deletion in existing deployments.

## Usage

### Creating a Cleanup Manager

```go
config := cleanup.Config{
    Enabled:         true,
    TTL:             24 * time.Hour,
    ScanInterval:    1 * time.Hour,
    BatchSize:       100,
    WorkerCount:     4,
    AdvisoryLockID:  0x74746b636c65616e,
    InstanceID:      "cleanup-instance-1",
}

manager := cleanup.NewManager(
    logger,
    config,
    storage,        // TokenStore with cleanup methods
    identityService, // For SKI derivation
    keystore,       // For key deletion
)

// Start cleanup
if err := manager.Start(); err != nil {
    return err
}

// Stop cleanup
manager.Stop()
```

## Cleanup Process Flow

1. Manager acquires leadership lock (PostgreSQL advisory lock)
2. Manager queries for deleted tokens older than TTL that haven't been cleaned
3. Manager distributes tokens to worker pool
4. For each token:
   - Derive SKIs from owner identity
   - Delete keys from keystore using SKIs
   - Mark token as cleaned in database
5. Release leadership lock
6. Wait for scan interval and repeat

## Token Lifecycle States

A token transitions through the following states related to cleanup:

- **Active**: Token is unspent and in use
- **Deleted**: Token marked as deleted (`is_deleted=true`, `spent_at` set)
- **Eligible for Cleanup**: Deleted token older than TTL with `keys_cleaned_at=NULL`
- **Cleaned**: Keys deleted from keystore (`keys_cleaned_at` set)

The cleanup service only processes tokens in the "Eligible for Cleanup" state.

## Database Schema

The cleanup service extends the tokens table with a new column:

```sql
ALTER TABLE tokens ADD COLUMN keys_cleaned_at TIMESTAMP;
```

This column tracks when a token's keys were cleaned from the keystore, preventing reprocessing.

## Distributed Deployment

### PostgreSQL
- **Multi-Instance Support**: Uses advisory locks for distributed coordination
- **Leader Election**: Only one replica performs cleanup sweeps at a time
- **High Availability**: Multiple replicas can share the same database
- **Automatic Failover**: If leader fails, another replica acquires leadership

### SQLite
- **Single-Node Only**: SQLite lacks advisory lock mechanism
- **Node Restart Support**: Cleanup resumes automatically after restart
- **Not Recommended**: For multi-replica deployments, use PostgreSQL

## Configuration Guidelines

### Default Values
- **TTL**: 24 hours (ensures tokens are truly finalized)
- **Scan Interval**: 1 hour (less aggressive than recovery's 5 seconds)
- **Batch Size**: 100 tokens per sweep
- **Worker Count**: 4 parallel workers
- **Advisory Lock ID**: `0x74746b636c65616e` ("ttkclean" in hex)

### Tuning Recommendations

1. **For High-Volume Environments:**
   - Increase `batchSize` to 200-500 for more tokens per sweep
   - Increase `workerCount` to 8-16 for faster parallel processing
   - Decrease `scanInterval` to 30m for more frequent cleanup

2. **For Resource-Constrained Systems:**
   - Decrease `workerCount` to 2 to reduce CPU usage
   - Increase `scanInterval` to 2-4h to reduce database load
   - Keep default `batchSize` to limit memory usage

3. **For Security-Sensitive Deployments:**
   - Decrease `ttl` to 12h for faster key removal
   - Decrease `scanInterval` to 30m for more frequent cleanup
   - Monitor cleanup metrics to ensure timely processing

4. **For Multi-Instance Deployments:**
   - **PostgreSQL Required**: Multi-instance deployments require PostgreSQL for distributed coordination
   - Keep default `advisoryLockID` unless running multiple independent cleanup systems
   - Consider setting explicit `instanceID` values for easier debugging and monitoring

### Performance Considerations
- Each scan queries the token database, so `scanInterval` directly affects database load
- `workerCount` affects CPU utilization during cleanup sweeps
- `batchSize` affects memory usage and the duration of each cleanup sweep
- The relationship `scanInterval < ttl` ensures timely cleanup without premature processing

## Error Handling

The cleanup service handles errors gracefully:

- **Key Not Found**: Logged as warning, continues with other keys
- **Partial Failures**: If some keys fail to delete, token is not marked as cleaned
- **Database Errors**: Logged and retried on next scan
- **Leadership Loss**: Cleanup stops gracefully, another instance takes over

## Monitoring

Key metrics to monitor:

- **Cleanup Rate**: Tokens cleaned per hour
- **Backlog Size**: Number of tokens eligible for cleanup
- **Error Rate**: Failed cleanup attempts
- **Leadership Changes**: Frequency of leader election
- **Processing Time**: Duration of each cleanup sweep

## Security Considerations

- **TTL Safety**: 24-hour default ensures tokens are finalized before key deletion
- **Idempotency**: Safe to retry cleanup operations
- **Audit Trail**: `keys_cleaned_at` timestamp provides cleanup history
- **Key Isolation**: Only deletes keys for deleted tokens, never active tokens

## Comparison with Recovery Service

| Feature | Recovery Service | Cleanup Service |
|---------|-----------------|-----------------|
| **Purpose** | Re-register finality listeners | Delete stale cryptographic keys |
| **Frequency** | Every 5 seconds | Every 1 hour |
| **TTL** | 30 seconds | 24 hours |
| **Target** | Pending transactions | Deleted tokens |
| **Urgency** | High (affects finality) | Low (housekeeping) |
| **Batch Size** | 100 | 100 |
| **Workers** | 4 | 4 |

## See Also

- [Storage Service](storage.md) - Database operations and interfaces
- [Configuration Guide](../configuration.md) - Detailed configuration parameters
- [Transaction Recovery Service](recovery.md) - Related recovery mechanism