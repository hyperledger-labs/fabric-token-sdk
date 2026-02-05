## The Token Selector Service

The Token Selector Service is responsible for selecting which tokens to use when assembling token transactions. It mitigates the risk of double-spending by implementing strategic token selection algorithms with built-in locking mechanisms.

The selector service is located under `token/services/selector`.

### Overview

When a token transaction needs to be created (e.g., a transfer of 100 USD), the selector must choose which specific tokens from the wallet to use. This selection process must:
- Find sufficient tokens to meet the requested amount
- Lock selected tokens to prevent concurrent transactions from using them
- Handle concurrent access from multiple processes
- Optimize for performance and memory usage

### Token Selection Strategies

The Token SDK supports multiple fetcher strategies that determine how tokens are loaded and cached:

#### 1. Lazy Fetcher

**Approach**: Query the database on every request.

**Characteristics**:
- Low memory usage (no caching)
- Always returns fresh data
- High database load
- Slower response times (DB query per request)

**Implementation**: `LazyFetcher` in `token/services/selector/sherdlock/fetcher.go`

#### 2. Eager/Cached Fetcher

**Approach**: Pre-load all tokens into memory and serve from cache.

**Characteristics**:
- Fast response times (cache hits)
- Low database load (periodic refresh only)
- High memory usage (loads ALL wallets)
- Wastes resources on unused wallets

**Cache Refresh**:
- Time-based: Every 1 second
- Usage-based: After 5 queries

**Implementation**: `cachedFetcher` in `token/services/selector/sherdlock/fetcher.go`

#### 3. Mixed Fetcher

**Approach**: Combines eager and lazy strategies - uses eager cache first, falls back to lazy if cache miss.

**Characteristics**:
- First request: Uses eager cache (fast if tokens exist)
- Cache miss: Falls back to lazy fetcher
- Subsequent requests: May use either strategy

**Implementation**: `mixedFetcher` in `token/services/selector/sherdlock/fetcher.go`

#### 4. Adaptive Fetcher (Recommended)

**Approach**: Hybrid strategy combining lazy discovery with eager refresh for known wallets only.

**Characteristics**:
- Memory efficient (only caches actively used wallets)
- Fast performance (cache hits for known wallets)
- Scalable (handles thousands of wallets)
- Self-regulating (optional TTL for inactive wallets)
- Adaptive (automatically adjusts to usage patterns)

**Implementation**: `adaptiveFetcher` in `token/services/selector/sherdlock/fetcher.go`

See [Adaptive Cache Strategy](#adaptive-cache-strategy) below for detailed explanation.

### Strategy Comparison

| Feature | Lazy | Eager/Cached | Mixed | **Adaptive** |
|---------|------|--------------|-------|--------------|
| Memory Usage | Low | High | High | **Medium (adaptive)** |
| DB Load | High | Medium | Medium | **Low (targeted)** |
| First Request | Slow | Fast | Fast | Slow |
| Subsequent | Slow | Fast | Fast | **Fast** |
| Unused Wallets | N/A | Cached | Cached | **Not cached** |
| TTL Support | N/A | No | No | **Yes** |
| Production Ready | Limited | Yes | No | **Yes (Recommended)** |

### Configuration

Strategies are configured through the `FetcherProvider`:

```go
// Create selector with adaptive strategy (recommended)
provider := NewFetcherProvider(
    storeServiceManager,
    notifierManager,
    metricsProvider,
    Adaptive,  // Strategy: Lazy, Eager, Mixed, or Adaptive
)
```

### Adaptive Cache Strategy

The Adaptive Fetcher implements a sophisticated caching mechanism optimized for production environments.

#### Overview

**Hybrid approach**: Lazy discovery + Eager refresh for known wallets only

#### Key Concept: Known Wallets

**Known Wallet Definition:** A wallet that the application has requested tokens for at least once. Known wallets are tracked in a `knownWallets map[string]*walletInfo` data structure. A wallet becomes known when it is first requested (cache miss) and stays known until:
- It expires due to TTL (if TTL > 0)
- The application restarts

The purpose of tracking known wallets is to refresh only the wallets the application actually uses, rather than all wallets in the system.

#### Policies

**Refresh Triggers**:
1. **Time-based**: Every 1 second (`freshnessInterval`)
2. **Usage-based**: After 5 queries (`maxQueriesBeforeRefresh`)

**Refresh Types**:
1. **Hard Refresh**: Cache stale (>1 sec) → Blocking update
2. **Soft Refresh**: Cache overused (>5 queries) → Background update

**Wallet TTL (Optional)**:
- **TTL = 0** (default): Wallets never expire from cache
- **TTL > 0**: Wallets not accessed within TTL duration are removed during refresh
- Each wallet tracks `lastAccess` time, updated on every request
- Expired wallets can be rediscovered if accessed again

#### How It Works

**First Request (Cache Miss)**:

```
Transfer requests wallet X, type USD
↓
Check cache for key "wallet_X:USD"
↓
Cache miss (wallet X not in cache)
↓
Lazy load from database:
  Query: SpendableTokensIteratorBy(ctx, "wallet_X", "USD")
  Load ONLY wallet X's USD tokens (not all system tokens)
↓
Add to cache:
  cache["wallet_X:USD"] = tokens
↓
Mark wallet X as known:
  knownWallets["wallet_X"] = {lastAccess: now()}
↓
Return result (1-5ms)
```

**Subsequent Requests (Cache Hit)**:

```
Transfer requests wallet X, type USD
↓
Check cache for key "wallet_X:USD"
↓
Cache hit (wallet X found in cache)
↓
Update last access time:
  knownWallets["wallet_X"].lastAccess = now()
↓
Return cached result immediately
↓
Instant response (<1 µs)
```

**Periodic Refresh**:

```
Refresh triggered (time-based OR usage-based)
↓
Iterate through knownWallets map:
  For each walletID in knownWallets:
    ↓
    Check TTL expiration (if TTL > 0):
      If now - lastAccess > TTL:
        → Skip this wallet (will be removed)
        → Continue to next wallet
    ↓
    Query: SpendableTokensIteratorBy(ctx, walletID, "")
    ↓
    Get ALL tokens for this wallet (all currency types)
    ↓
    Group tokens by currency type:
      wallet_X:USD → [token1, token2, ...]
      wallet_X:EUR → [token3, token4, ...]
    ↓
    Update cache with fresh data:
      cache["wallet_X:USD"] = fresh_usd_tokens
      cache["wallet_X:EUR"] = fresh_eur_tokens
↓
Remove expired wallets from knownWallets (if TTL > 0)
↓
Reset timer and query counter:
  lastFetched = now()
  queriesResponded = 0
```

#### Example Scenario

**System with 1,000 wallets, application uses only 10**:

```
Time 0:00 - Application starts
  knownWallets = {}  (empty)
  cache = {}  (empty)

Time 0:01 - First transfer from wallet_alice (USD)
  → Cache miss
  → Query DB for wallet_alice USD tokens only
  → knownWallets = {"wallet_alice": {lastAccess: 0:01}}
  → cache = {"wallet_alice:USD": [tokens]}

Time 0:02 - Second transfer from wallet_alice (USD)
  → Cache hit (instant response)
  → Update lastAccess timestamp

Time 0:03 - Refresh triggered (1 second passed)
  → Refresh only wallet_alice (not all 1,000 wallets!)
  → Update cache with fresh data

After 1 hour of operation:
  knownWallets = {10 wallets actually used}
  cache = {20 entries (10 wallets × 2 currencies avg)}
  Memory: 2.9 MB (vs 290 MB if all 1,000 wallets were cached)
  Refresh time: 11.5 ms (vs 1,150 ms for all wallets)
```

#### TTL Example (with 10-minute TTL)

```
Time 10:00 - Three wallets known
  wallet_alice: lastAccess = 10:00
  wallet_bob: lastAccess = 10:00
  wallet_charlie: lastAccess = 10:00

Time 10:05 - Access alice and bob
  wallet_alice: lastAccess = 10:05 (updated)
  wallet_bob: lastAccess = 10:05 (updated)
  wallet_charlie: lastAccess = 10:00 (not accessed)

Time 10:12 - Refresh triggered
  Check expiration (TTL = 10 min):
    wallet_alice: 10:12 - 10:05 = 7 min → Keep
    wallet_bob: 10:12 - 10:05 = 7 min → Keep
    wallet_charlie: 10:12 - 10:00 = 12 min → Expire
  
  Result:
    Refresh alice and bob
    Remove charlie from knownWallets and cache

Time 10:15 - Access charlie again
  Cache miss (expired)
  Lazy load from DB
  wallet_charlie: lastAccess = 10:15 (rediscovered)
```

#### Configuration

```go
const (
    freshnessInterval     = 1 * time.Second
    maxQueries            = 5
    noWalletTTLExpiration = 0  // Wallets never expire (default)
)

// Create adaptive fetcher (no TTL)
fetcher := newAdaptiveFetcher(
    tokenDB,
    freshnessInterval,
    maxQueries,
)

// Or with custom TTL (10 minutes)
fetcher := newAdaptiveFetcherWithTTL(
    tokenDB,
    freshnessInterval,
    maxQueries,
    10 * time.Minute,  // Expire wallets after 10 minutes of inactivity
)
```

**Monitoring**:
- Track cache hit rates
- Monitor memory usage
- Measure database query frequency
- Observe token selection latency

The selector service exposes metrics through the `Metrics` interface for monitoring and optimization.
