# Transaction Recovery

## Overview

The Transaction Recovery feature ensures that token transactions can be recovered and completed even if a replica crashes after submitting a transaction for ordering but before receiving the finality event from the blockchain network.

This is particularly important in high-availability deployments where multiple replicas share the same PostgreSQL database.

## How It Works

1. **Transaction Tracking**: When a transaction is prepared and sent for ordering, it's stored in the TTXDB with status "Pending"
2. **Background Scanning**: A recovery manager periodically scans for pending transactions older than a configured TTL
3. **Listener Re-registration**: For each recovered transaction, the finality listener is re-registered to receive status updates
4. **Duplicate Prevention**: An in-memory set tracks recovered transactions to avoid duplicate processing
5. **Graceful Shutdown**: The recovery manager stops cleanly when the network service shuts down

## Configuration

Transaction recovery is configured per TMS (Token Management Service) in your configuration file:

```yaml
token:
  tms:
    mytms:  # Your TMS ID
      network:
        recovery:
          enabled: true           # Enable/disable recovery (default: false)
          ttl: 30s               # Time before a pending tx is considered for recovery (default: 30s)
          scanInterval: 30s      # How often to scan for pending transactions (default: 30s)
```

### Configuration Parameters

- **enabled** (bool): Whether transaction recovery is enabled
  - Default: `false` (disabled for backward compatibility)
  - Set to `true` to enable recovery in multi-replica deployments

- **ttl** (duration): Time-to-live for pending transactions
  - Default: `30s`
  - Transactions pending longer than this duration are considered for recovery
  - Should be longer than your typical transaction finality time
  - Recommended: 30s-60s for most deployments

- **scanInterval** (duration): How often to scan for pending transactions
  - Default: `30s`
  - Lower values provide faster recovery but increase database load
  - Higher values reduce overhead but delay recovery
  - Recommended: Same as TTL or slightly longer

## Use Cases

### Multi-Replica Deployment

In a deployment with multiple replicas sharing a PostgreSQL database:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Replica 1  │     │  Replica 2  │     │  Replica 3  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────▼──────┐
                    │  PostgreSQL │
                    │   (Shared)  │
                    └─────────────┘
```

**Scenario**: Replica 1 prepares a transaction and sends it for ordering, then crashes before receiving the finality event.

**Without Recovery**: The transaction remains in "Pending" state indefinitely. Users must manually intervene.

**With Recovery**: Replica 2 or 3 detects the pending transaction after the TTL expires, re-registers the finality listener, and completes the transaction lifecycle when the finality event arrives.

## Best Practices

1. **Enable in Production**: Always enable recovery in multi-replica production deployments
2. **Tune TTL**: Set TTL based on your network's typical finality time plus a safety margin
3. **Monitor Logs**: Watch for recovery events in logs to detect replica failures
4. **Database Performance**: Ensure your database can handle the periodic scans efficiently
5. **Test Failover**: Regularly test replica failover scenarios to verify recovery works

## Monitoring

The recovery manager logs important events:

```
INFO  Starting transaction recovery manager for TMS [mytms]
INFO  Recovered transaction [txid123] - re-registered finality listener
WARN  Failed to recover transaction [txid456]: error details
INFO  Stopping transaction recovery manager for TMS [mytms]
```

Monitor these logs to:
- Detect when replicas crash (recovery events increase)
- Identify problematic transactions that fail to recover
- Verify the recovery system is functioning correctly

## Limitations

1. **Network Dependency**: Recovery only works if at least one replica remains running
2. **Database Requirement**: Requires a shared database (PostgreSQL) across replicas
3. **Finality Events**: Can only recover transactions that will eventually receive finality events
4. **Memory Overhead**: Tracks recovered transaction IDs in memory (cleared on restart)

## Troubleshooting

### Recovery Not Working

**Symptom**: Transactions remain pending after replica failure

**Checks**:
1. Verify `enabled: true` in configuration
2. Check that TTL has elapsed since transaction was stored
3. Ensure at least one replica is running
4. Review logs for error messages
5. Verify database connectivity

### High Database Load

**Symptom**: Database CPU/IO spikes periodically

**Solution**:
1. Increase `scanInterval` to reduce scan frequency
2. Add database indexes on `status` and `stored_at` columns (should exist by default)
3. Consider increasing TTL to reduce the number of transactions scanned

### Duplicate Processing

**Symptom**: Same transaction recovered multiple times

**Note**: This should not happen due to in-memory tracking, but if it does:
1. Check for clock skew between replicas
2. Verify only one recovery manager runs per TMS per replica
3. Review logs for race conditions

## Example Configurations

### Conservative (Low Load)
```yaml
token:
  tms:
    mytms:
      network:
        recovery:
          enabled: true
          ttl: 60s
          scanInterval: 60s
```

### Aggressive (Fast Recovery)
```yaml
token:
  tms:
    mytms:
      network:
        recovery:
          enabled: true
          ttl: 15s
          scanInterval: 15s
```

### Disabled (Single Replica)
```yaml
token:
  tms:
    mytms:
      network:
        recovery:
          enabled: false
```

## See Also

- [Token Transaction Service (TTX)](ttx.md)
- [Storage Service](storage.md)
- [Network Service](network.md)