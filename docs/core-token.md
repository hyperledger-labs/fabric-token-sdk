# Example core.yaml section

The following example provides descriptions for the various keys required by the Token SDK.

```yaml
# ------------------- Token SDK Configuration -------------------------
token:
  # Is the version of this configuration structure. 
  # If not specified, the latest version is used 
  version: v1
  # Is the token-sdk enabled
  enabled: true

  # token selector configuration allows to use different implementations of the token selector
  # the "sherdlock" driver is the default implementation, other possible configurations are: "simple"
  # if empty, the default selector is used
  selector:
    driver: sherdlock
    # tokens might be locked because of an ongoing transaction from the same wallet. Instead of failing immediately, the selector can retry.
    # The interval is the exact amount of seconds (for simple selector) or the max amount of seconds (sherdlock) the selector waits before retrying.
    # default: 5s
    retryInterval: 5s
    # retry to gain a lock on tokens this amount of times before failing the transaction
    numRetries: 3
    # leaseExpiry is the period a token can be locked, after which it is forcefully unlocked
    # if leaseExpiry is zero, the eviction algorithm is never executed
    leaseExpiry: 3m
    # leaseCleanupTickPeriod defines how often the eviction algorithm must be executed
    # if leaseCleanupTickPeriod is zero, the eviction algorithm is never executed
    leaseCleanupTickPeriod: 90s
  # when we are interested to know when a tx reaches finality, we subscribe to the Finality Listener Manager for the finality event of that tx
  # this configuration specifies the way the manager is instantiated, i.e. how it gets notified about the finality events, how often it checks
  finality:
    # the implementation of the finality manager. The finality manager keeps track of all subscribers that are interested to a transaction.
    # These are the possible values:
    # delivery: The manager subscribes to the delivery service and receives all final transactions.
    # This manager keeps two structures: an LRU cache of recently finalized transactions, and a list of listeners that are waiting for future transactions.
    # When a new block comes from the delivery service, we store all new transactions in the cache and notify all interested listeners
    # When a client subscribes to the manager for a specific transaction, we go through the following steps, that correspond to all possible scenarios with decreasing probability:
    # a) The transaction reached finality recently, so we perform a lookup. If not found, we go to step b.
    # b) The transaction will reach finality shortly, so we append a listener and wait for a timeout. If the listener reaches timeout, we go to step c.
    # c) The transaction reached finality long ago, so we query the whole ledger for this specific transaction. If the query returns no result, we go to step d.
    # d) The transaction will reach finality at some point beyond the timeout or never, so we return Unknown. Then it is up to the client to either append another listener or accept that the transaction will never reach finality.  
    # committer: The manager subscribes to the commit pipeline and receives all finalized transactions.
    # If we subscribe for a transaction that hasn't been finalized yet, we will get notified once it reaches the commit pipeline.
    # For listeners that haven been invoked yet, either the transaction hasn't been finalized or it was finalized before we subscribed.
    # For this reason, there is a periodic polling period (1s) that queries all txIDs for which there is a pending listener and invokes the listeners for the ones found.
    # Once we get notified about finality, we try (repeatedly) to fetch the additional tx information needed from the vault (e.g. transactionRequest)
    # If we work without replicas, there is no need to try more than once to fetch the additional tx information, because the finality notification and vault update happen in sync.
    # It is only needed in case we have replicas, where another replica may have updated the vault shortly before.
    type: delivery
    # Only applicable when type = 'committer'
    committer:
      # how many times should we try to fetch the additional tx information from the vault
      maxRetries: 3
      # how long should we wait before retrying
      retryWaitDuration: 5s
    # Only applicable when type = 'delivery'
    delivery:
      # how many goroutines should process incoming transactions in parallel. Defaults to 1
      mapperParallelism: 10
      # how many blocks can we process in parallel when they arrive from the delivery service
      # if the value is <= 1, then blocks are processed sequentially.
      # This is the suggested configuration if we are not sure about the dependencies between blocks.
      # The total go routines processing transactions will be blockProcessParallelism * mapperParallelism
      blockProcessParallelism: 1
      # how many transactions should we guarantee that we keep in our recent cache
      # if the transaction is not among these elements we proceed to the step b, as described above.
      # if lruSize and lruBuffer are not set, then we will never evict past transactions (the cache will grow infinitely)
      lruSize: 30
      # eviction will happen when the cache size exceeds lruSize + lruBuffer
      lruBuffer: 15
      # when we can't find a transaction in the cache most probably it is about to become final
      # we will listen for this amount of time and then we will query the whole ledger, as described in step c.
      # if the timeout is not set, then the listener will never be evicted and we will never proceed to step c.
      # We will wait forever for the transaction to return (as is done for the 'committer' type)
      listenerTimeout: 10s
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, Orion, etc)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable

      # sections dedicated to the definition of the storage.
      # The Token-SDK uses multiple databases to keep track of transactions, tokens, identities, and audit records where it applies.  
      # These are the available databases:
      # ttxdb: store records of transactions 
      # tokendb: store information about the available tokens
      # auditdb: store audit records about the audited transactions
      # identitydb: store information about wallets and identities
      # The databases can be instantiated in isolation, a different backend for each db, or with a shared backend, depending on the driver used.
      # In the following example, we have all databases using the same backed but tokendb.

      # shared db configuration. The `unity` driver is used as provider.  
      db:
        persistence:
          # configuration for the unity db driver. It uses sql as backend. See also https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/core-fabric.md
          type: unity
          opts:
            driver: sqlite
            maxOpenConns: 20  # optional: max open read connections to the database. Defaults to unlimited. See https://go.dev/doc/database/manage-connections.
            maxIdleConns: 20  # optional: max idle read connections to the database. Defaults to 2.
            maxIdleTime: 30s  # optional: max duration a connection can be idle before it is closed. Defaults to 1 minute.
            dataSource: /some/path/unitydb
      # optional separate configuration for ttxdb, tokendb, auditdb, and identitydb
      tokendb:
        persistence:
          type: sql
          opts:
            driver: sqlite
            dataSource: /some/path/tokendb

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
              # Is this node an endorser?: true/false
              endorser: true
              # If this node is an endorser, which Fabric identity should be used to sign the endorsement?
              # If empty, the default identity will be used
              id:
              # This section is used to set the policy to be used to select the endorsers to contact.
              # Available policies are: `1outn`, `all`. Default policy is `all`
              policy:
                type: 1outn
              # A list of FSC node identifiers that must be contacted to obtain the endorsement 
              endorsers:
              - endorser1
              - endorser2
              - endorser2

      # sections dedicated to the definition of the wallets
      wallets:
        # Default cache size reference that can be used by any wallet that support caching
        defaultCacheSize: 3
        # owner wallets
        owners:
        - id: alice # the unique identifier of this wallet. Here is an example of use: `ttx.GetWallet(context, "alice")` 
          default: true # is this the default owner wallet
          # path to the folder containing the cryptographic material associated to wallet.
          # The content of the folder is driver dependent
          path:  /path/to/alice-wallet
          # Cache size, in case the wallet supports caching (e.g. idemix-based wallet)
          cacheSize: 3
        - id: alice.id1
          path: /path/to/alice.id1-wallet
        # issuer wallets
        issuers:
          - id: issuer # the unique identifier of this wallet. Here is an example of use: `ttx.GetIssuerWallet(context, "issuer)`
            default: true # is this the default issuer wallet
            # path to the folder containing the cryptographic material associated to wallet.
            # The content of the folder is driver dependent
            path: /path/to/issuer-wallet
            # additional options that can be used to instantiated the wallet.
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
          - id: auditor # the unique identifier of this wallet. Here is an example of use: `ttx.GetAuditorWallet(context, "auditor)`
            default: true # is this the default auditor wallet  
            # path to the folder containing the cryptographic material associated to wallet.
            # The content of the folder is driver dependent
            path: /path/to/auditor-wallet
            # additional options that can be used to instantiated the wallet.
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
