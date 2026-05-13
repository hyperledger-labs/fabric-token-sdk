# Selector Resource Limits - Security Guide

> **Performance Note**: Database-level LIMIT optimization has been implemented to reduce I/O overhead.


## Overview

The Fabric Token SDK selector service implements hard resource limits to prevent algorithmic attacks that could exhaust system resources (CPU, memory, storage) through maliciously crafted token selection requests.

## Threat Model

### Attack Vector

An attacker can craft token selection requests that:
1. **Iterate through millions of tokens** - Exhausting CPU and memory
2. **Attempt excessive lock acquisitions** - Creating lock contention storms
3. **Trigger infinite retry loops** - Blocking the system indefinitely
4. **Accumulate unbounded locks** - Exhausting lock storage
5. **Consume unlimited wall-clock time** - Preventing other operations

### Why This Matters

Token selection happens **before blockchain consensus**, meaning:
- **No transaction fees** - Attacker doesn't pay for failed selections
- **Pure DoS vector** - Can exhaust resources without valid transactions
- **Amplification** - One malicious request → millions of operations
- **Cascading failure** - Resource exhaustion affects all users
- **Limited audit trail** - Failed selections may not be comprehensively logged

## Resource Limits

### 1. Token Iteration Depth Limit

**What it limits**: Maximum number of tokens examined during selection

**Default**: 10,000 tokens

**Configuration**:
```yaml
token:
  selector:
    limits:
      maxTokensPerSelection: 10000
```

**When it triggers**: When the selector examines more than the configured number of tokens

**Error message**: `"token selection aborted: exceeded max token iteration limit (10000 tokens)"`

**Database optimization**: This limit is enforced at **two levels**:
1. **Database query level**: SQL queries include `LIMIT ?` clause to prevent fetching excess rows
2. **Application level**: Iterator counter provides defense-in-depth safety check

**Performance impact**: Database-level LIMIT significantly reduces I/O overhead:
- Without LIMIT: Database may scan millions of rows, returning them one-by-one
- With LIMIT: Database stops after returning configured number of rows
- Result: ~10x faster query execution for large token sets

**Tuning guidance**:
- **Increase** if legitimate operations require examining more tokens
- **Decrease** for tighter security in low-throughput environments
- **Monitor** actual token counts in production before adjusting

### 2. Lock Acquisition Attempt Limit

**What it limits**: Maximum number of lock operations attempted during selection

**Default**: 50,000 attempts (5x token iteration limit)

**Configuration**:
```yaml
token:
  selector:
    limits:
      maxLockAttempts: 50000
```

**When it triggers**: When the selector attempts more lock operations than configured

**Error message**: `"token selection aborted: exceeded max lock attempts (50000) after examining X tokens"`

**Tuning guidance**:
- Should be ≥ `maxTokensPerSelection` (validation enforced)
- Higher values allow for more lock contention tolerance
- Set to 5-10x `maxTokensPerSelection` for high-concurrency environments

### 3. Retry Cycle Limit

**What it limits**: Maximum number of outer retry loops during selection

**Default**: 10 cycles

**Configuration**:
```yaml
token:
  selector:
    limits:
      maxRetryCycles: 10
```

**When it triggers**: When the selector retries more times than configured

**Error message**: `"token selection aborted: exceeded max retry cycles (10) after examining X tokens and Y lock attempts"`

**Tuning guidance**:
- **Increase** in high-contention environments where retries are common
- **Decrease** for faster failure in low-contention environments
- Balance between resilience and attack mitigation

### 4. Lock Store Growth Limit

**What it limits**: Maximum number of locks a single transaction can hold

**Default**: 5,000 locks

**Configuration**:
```yaml
token:
  selector:
    limits:
      maxLocksPerTransaction: 5000
```

**When it triggers**: When a transaction tries to acquire more locks than configured

**Error message**: `"lock limit exceeded: transaction TX already holds 5000 locks (max: 5000)"`

**Tuning guidance**:
- Should be ≤ `maxTokensPerSelection` (validation enforced)
- **Increase** for bulk operations that legitimately need many locks
- **Decrease** to reduce memory footprint per transaction

### 5. Wall-Clock Timeout

**What it limits**: Maximum time allowed for entire selection operation

**Default**: 30 seconds

**Configuration**:
```yaml
token:
  selector:
    limits:
      selectionTimeout: 30s
```

**When it triggers**: When selection takes longer than configured timeout

**Error message**: `"token selection aborted: exceeded timeout (30s) after examining X tokens and Y lock attempts"`

**Tuning guidance**:
- **Increase** for slow databases or bulk operations
- **Decrease** for faster failure detection
- Consider database query performance when setting

## Configuration Examples

### Default Secure Configuration

```yaml
token:
  selector:
    limits:
      maxTokensPerSelection: 10000      # 10k tokens ≈ 1MB memory
      maxLockAttempts: 50000            # 5x iteration limit
      maxRetryCycles: 10                # Reasonable for transient issues
      maxLocksPerTransaction: 5000      # Half of iteration limit
      selectionTimeout: 30s             # Generous for legitimate use
```

### High-Throughput Environment

```yaml
token:
  selector:
    limits:
      maxTokensPerSelection: 50000      # More tokens for bulk ops
      maxLockAttempts: 250000           # Proportionally higher
      maxRetryCycles: 20                # More retries for high contention
      maxLocksPerTransaction: 25000     # Proportionally higher
      selectionTimeout: 120s            # Longer timeout for bulk ops
```

### Low-Latency Environment

```yaml
token:
  selector:
    limits:
      maxTokensPerSelection: 5000       # Fewer tokens for faster failure
      maxLockAttempts: 25000            # Proportionally lower
      maxRetryCycles: 5                 # Fewer retries for faster failure
      maxLocksPerTransaction: 2500      # Proportionally lower
      selectionTimeout: 10s             # Shorter timeout
```

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Limit Violations**
   - Track frequency of each limit type being hit
   - Alert on sudden increases in violations

2. **Resource Usage**
   - Monitor actual tokens examined per selection
   - Track lock attempt counts
   - Measure selection duration

3. **Success Rate**
   - Track ratio of successful vs. aborted selections
   - Alert on drops in success rate

### Recommended Alerts

```yaml
# Alert when limit violations exceed threshold
- alert: HighSelectorLimitViolations
  expr: rate(selector_limit_violations_total[5m]) > 10
  annotations:
    summary: "High rate of selector limit violations"
    description: "{{ $value }} limit violations per second"

# Alert when selection success rate drops
- alert: LowSelectorSuccessRate
  expr: rate(selector_success_total[5m]) / rate(selector_attempts_total[5m]) < 0.9
  annotations:
    summary: "Selector success rate below 90%"
```

## Operational Procedures

### Responding to Limit Violations

1. **Investigate the cause**
   - Check logs for patterns in aborted selections
   - Identify if it's legitimate load or attack

2. **Temporary mitigation**
   - If legitimate: Increase relevant limits
   - If attack: Block malicious actors at network level

3. **Long-term resolution**
   - Optimize token distribution to reduce selection complexity
   - Implement rate limiting at application level
   - Consider sharding or partitioning strategies

### Tuning Process

1. **Baseline measurement**
   - Run in production with default limits
   - Collect metrics for 1-2 weeks

2. **Analysis**
   - Calculate 99th percentile for each resource
   - Add 50% safety margin

3. **Gradual adjustment**
   - Increase limits incrementally
   - Monitor impact on system resources
   - Validate no degradation in performance

4. **Documentation**
   - Document rationale for custom limits
   - Record baseline metrics used for tuning

## Security Best Practices

1. **Never disable limits** - All limits are enforced by default
2. **Start conservative** - Use default values initially
3. **Monitor continuously** - Track metrics and violations
4. **Tune based on data** - Use real production metrics for adjustments
5. **Document changes** - Record why limits were adjusted
6. **Review regularly** - Reassess limits as usage patterns change
7. **Test thoroughly** - Validate limits in staging before production

## Validation Rules

The configuration system enforces these relationships:

- `maxLockAttempts` ≥ `maxTokensPerSelection`
- `maxLocksPerTransaction` ≤ `maxTokensPerSelection`
- All limits must be positive integers
- Timeout must be positive duration

Invalid configurations will be rejected at startup with clear error messages.

## Migration Guide

### For Existing Deployments

1. **Review current usage**
   - Analyze logs for typical selection patterns
   - Identify maximum tokens selected in legitimate operations

2. **Test in staging**
   - Deploy with default limits
   - Run full test suite
   - Monitor for limit violations

3. **Adjust if needed**
   - Increase limits only if legitimate operations are blocked
   - Document rationale for any increases

4. **Deploy to production**
   - Roll out gradually (canary → full deployment)
   - Monitor closely for first 24-48 hours
   - Be prepared to adjust limits if needed

### Breaking Changes

This implementation enforces limits by default. Existing deployments must:
- Review and potentially adjust limits before upgrading
- Test thoroughly in non-production environments
- Plan for potential operational impact

## FAQ

**Q: Can I disable these limits?**
A: No. Limits are always enforced for security. You can increase them if needed.

**Q: What happens to locks when selection is aborted?**
A: All acquired locks are automatically released via `UnlockByTxID`.

**Q: Will this affect my existing transactions?**
A: Only if they exceed the default limits. Test in staging first.

**Q: How do I know if limits are too restrictive?**
A: Monitor limit violation metrics and selection success rates.

**Q: Can limits be different per TMS?**
A: Currently no, limits are global. This may be added in future versions.

## References

- [Selector Service Documentation](../services/selector.md)
- [Security Plan](../../SECURITY_PLAN_SELECTOR_LIMITS.md)
- [Configuration Guide](../configuration.md)