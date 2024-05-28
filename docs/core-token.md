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
  # the "default" driver is the mailman implementation, other possible configurations are: "simple"
  selector:
    driver: mailman

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
          # configuration for the unity db driver. It uses sql as backend
          type: unity
          opts:
            createSchema: true
            driver: sqlite
            maxOpenConns: 10
            dataSource: /some/path/unitydb
      tokendb:
        persistence:
          type: sql
          opts:
            createSchema: true 
            driver: sqlite    
            maxOpenConns: 10
            dataSource: /some/path/tokendb

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
