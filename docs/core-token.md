# Example core.yaml section

The following example provides descriptions for the various keys required by the Token SDK.

```yaml
# ------------------- Token SDK Configuration -------------------------
token:
  # version is the version of this configuration structure. 
  # If not specified, the latest version is used.
  version: v1
  # enabled determines if the Token SDK is enabled.
  enabled: true

  # selector configuration allows the use of different implementations of the token selector.
  # The "sherdlock" driver is the default implementation; other possible configurations are: "simple".
  # If empty, the default selector is used.
  selector:
    driver: sherdlock
    # tokens might be locked because of an ongoing transaction from the same wallet. Instead of failing immediately, the selector can retry.
    # The interval is the exact amount of seconds (for simple selector) or the max amount of seconds (sherdlock) the selector waits before retrying.
    # User-defined default: 5s
    retryInterval: 5s
    # numRetries is the number of times to retry gaining a lock on tokens before failing the transaction.
    numRetries: 3
    # leaseExpiry is the period a token can be locked, after which it is forcefully unlocked.
    # If leaseExpiry is zero, the eviction algorithm is never executed.
    leaseExpiry: 3m
    # leaseCleanupTickPeriod defines how often the eviction algorithm must be executed.
    # If leaseCleanupTickPeriod is zero, the eviction algorithm is never executed.
    leaseCleanupTickPeriod: 90s

  # When we are interested in knowing when a transaction reaches finality, we subscribe to the Finality Listener Manager for the finality event of that transaction.
  # This configuration specifies the way the manager is instantiated (i.e., how it gets notified about the finality events, how often it checks).
  finality:
    # driver is the implementation of the finality manager. The finality manager keeps track of all subscribers that are interested in a transaction.
    # These are the possible values:
    # delivery: The manager subscribes to the delivery service and receives all final transactions.
    #   This manager keeps two structures: an LRU cache of recently finalized transactions, and a list of listeners that are waiting for future transactions.
    #   When a new block comes from the delivery service, we store all new transactions in the cache and notify all interested listeners.
    #   When a client subscribes to the manager for a specific transaction, we go through the following steps, which correspond to all possible scenarios with decreasing probability:
    #   a) The transaction reached finality recently, so we perform a lookup. If not found, we proceed to step b.
    #   b) The transaction will reach finality shortly, so we append a listener and wait for a timeout. If the listener reaches timeout, we proceed to step c.
    #   c) The transaction reached finality long ago, so we query the whole ledger for this specific transaction. If the query returns no result, we proceed to step d.
    #   d) The transaction will reach finality at some point beyond the timeout or never, so we return Unknown. Then it is up to the client to either append another listener or accept that the transaction will never reach finality.  
    # notification: The manager is notified about finality events via a notification service (e.g. for FabricX).
    #   When a new notification arrives, an event is added to a queue for asynchronous processing.
    #   When a client subscribes to the manager for a specific transaction, we perform an immediate query to check its status.
    # The field can also be left empty. In that case, the default option will be used depending on the network type each TMS refers to.
    type: delivery
    # Only applicable when type = 'delivery'
    delivery:
      # mapperParallelism is the number of goroutines that process incoming transactions in parallel. Defaults to 1.
      mapperParallelism: 10
      # blockProcessParallelism is the number of blocks we can process in parallel when they arrive from the delivery service.
      # If the value is <= 1, then blocks are processed sequentially.
      # This is the suggested configuration if we are not sure about the dependencies between blocks.
      # The total go routines processing transactions will be blockProcessParallelism * mapperParallelism.
      blockProcessParallelism: 1
      # lruSize detects how many transactions we should guarantee to keep in our recent cache.
      # If the transaction is not among these elements, we proceed to step b, as described above.
      # If lruSize and lruBuffer are not set, then we will never evict past transactions (the cache will grow infinitely).
      lruSize: 30
      # eviction will happen when the cache size exceeds lruSize + lruBuffer.
      lruBuffer: 15
      # listenerTimeout is the duration to listen when we can't find a transaction in the cache (most probably it is about to become final).
      # We will listen for this amount of time and then we will query the whole ledger, as described in step c.
      # If the timeout is not set, then the listener will never be evicted and we will never proceed to step c.
      # We will wait forever for the transaction to return (as is done for the 'committer' type).
      listenerTimeout: 10s
    # Only applicable when type = 'notification'
    notification:
      # workers is the number of goroutines that process events in parallel. Defaults to 10.
      workers: 10
      # queueSize is the size of the event buffer. Defaults to 1000.
      queueSize: 1000
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, etc.)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable

      # sections dedicated to the definition of the storage.
      # The Token SDK uses multiple databases to keep track of transactions, tokens, identities, and audit records where applicable.
      # These are the available databases:
      # ttxdb: stores records of transactions.
      # tokendb: stores information about the available tokens.
      # auditdb: stores audit records about the audited transactions.
      # identitydb: stores information about wallets and identities.
      # The databases can be instantiated in isolation, a different backend for each db, or with a shared backend, depending on the driver used.
      # In the following example, we have all databases using the same backend but tokendb.

      # optional separate configuration for ttxdb, tokendb, tokenlockdb, auditdb, and identitydb
      # otherwise they default to 'default', if it is defined
      tokendb:
        persistence: my_token_persistence

      services:
        # This section contains network specific configuration
        network:
          # Configuration related to the Fabric network
          fabric:
            # In Fabric, the execution of the token chaincode can be endorsed by any node equipped with
            # a proper endorsement key.
            # Therefore, also FSC nodes equipped with proper endorsement keys can perform the same function.
            # This section is dedicated to the configuration of the endorsement of the token chaincode by
            # other FSC nodes.
            fsc_endorsement:
              # Is this node an endorser? true/false
              endorser: true
              # If this node is an endorser, which Fabric identity should be used to sign the endorsement?
              # If empty, the default identity will be used.
              id:
              # This section is used to set the policy to be used to select the endorsers to contact.
              # Available policies are: `1outn`, `all`. Default policy is `all`.
              policy:
                type: 1outn
              # A list of FSC node identifiers that must be contacted to obtain the endorsement.
              endorsers:
              - endorser1
              - endorser2
              - endorser2

      # sections dedicated to the definition of the wallets
      wallets:
        # Default cache size reference that can be used by any wallet that supports caching.
        defaultCacheSize: 3
        # owner wallets
        owners:
        - id: alice # the unique identifier of this wallet. Here is an example of use: `ttx.GetWallet(context, "alice")` 
          default: true # is this the default owner wallet
          # path to the folder containing the cryptographic material associated with the wallet.
          # The content of the folder is driver dependent.
          path:  /path/to/alice-wallet
          # Cache size, in case the wallet supports caching (e.g., idemix-based wallet).
          cacheSize: 3
        - id: alice.id1
          path: /path/to/alice.id1-wallet
        # issuer wallets
        issuers:
          - id: issuer # the unique identifier of this wallet. Here is an example of use: `ttx.GetIssuerWallet(context, "issuer")`
            default: true # is this the default issuer wallet
            # path to the folder containing the cryptographic material associated with the wallet.
            # The content of the folder is driver dependent.
            path: /path/to/issuer-wallet
            # additional options that can be used to instantiate the wallet.
            # options are driver dependent. With `fabtoken` and `dlog` drivers,
            # the following options apply.
            opts:
              BCCSP:
                Default: SW
                SW:
                  Hash: SHA2
                  Security: 256
                # The following only needs to be defined if the BCCSP Default is set to PKCS11.
                # NOTE: in order to use pkcs11, you have to build the application with "go build -tags pkcs11"
                PKCS11:
                  Hash: SHA2
                  Label: null
                  Library: null
                  Pin: null
                  Security: 256
        # auditor wallets
        auditors:
          - id: auditor # the unique identifier of this wallet. Here is an example of use: `ttx.GetAuditorWallet(context, "auditor")`
            default: true # is this the default auditor wallet  
            # path to the folder containing the cryptographic material associated with the wallet.
            # The content of the folder is driver dependent.
            path: /path/to/auditor-wallet
            # additional options that can be used to instantiate the wallet.
            # options are driver dependent. With `fabtoken` and `dlog` drivers,
            # the following options apply
            opts:
              BCCSP:
                Default: SW
                PKCS11:
                  Hash: SHA2
                  Label: null
                  Library: null
                  Pin: null
                  Security: 256
                SW:
                  Hash: SHA2
                  Security: 256
```

# Configuration Defaults and Optional Sections

## Optional: token.selector

If not specified, the default selector implementation is used.

Default values:
- driver: sherdlock
- retryInterval: 5s
- numRetries: 3
- leaseExpiry: 3m
- leaseCleanupTickPeriod: 90s

---

## Optional: token.finality

If not specified, the default configuration is:

```yaml
token:
  finality:
    type: delivery
```

Default values:

- type: delivery
- committer.maxRetries: 3
- committer.retryWaitDuration: 5s
- delivery.mapperParallelism: 10
- delivery.blockProcessParallelism: 10
- delivery.lruSize: 30
- delivery.lruBuffer: 15
- delivery.listenerTimeout: 10s

# Minimal Valid Token SDK Configuration

The Token SDK can start with minimal configuration because sensible defaults are applied.

Example:

```yaml
token:
  enabled: true
```

All other sections will use default values unless explicitly configured.