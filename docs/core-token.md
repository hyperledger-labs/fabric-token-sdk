# Example core.yaml section

The following example provides descriptions for the various keys required for a Fabric Smart Client node that uses the Token SDK.

```yaml
# ------------------- Token SDK Configuration -------------------------
token:
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
      # the name of the driver that provides the implementation of the Driver API.
      # This field is optional. If not specified, the Token-SDK will derive this information by fetching the public parameters
      # from the remote network
      driver: zkatdlog 
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
  # Internal database to keep track of token transactions. 
  # It is used by auditors and token owners to track history
  ttxdb:
    persistence:
      # type can be one of badger, sql or memory.
      type: badger
      opts:
        # persistence location
        path: /some/path

    # The sql driver uses golangs database/sql package internally.
    # In theory you can use any driver if you import it in your application;
    # for instance `import _ "github.com/mattn/go-sqlite3"` for the cgo version of sqlite.
    # See https://github.com/golang/go/wiki/SQLDrivers. We only tested with github.com/lib/pq
    # and modernc.org/sqlite, and it's likely that other drivers don't work exactly the same.
    # To try a new sql driver, add a test here: token/services/ttxdb/db/sql/sql_test.go.
    # 
    # type: sql
    # opts:
    #   createSchema: true # create tables programmatically
    #   tablePrefix: tsdk  # optional
    #   driver: sqlite     # in the application, `import _ "modernc.org/sqlite"`
    #   dataSource: /some/path/ttxdb.sqlite
    #
    # Alternative (`import _ "github.com/lib/pq"`).
    # The 'dataSource' field can be sensitive (contain a password). In that case,
    # set it in the TTXDB_DATASOURCE environment variable instead of in this file.
    #   driver: postgres
    #   dataSource: host=localhost port=5432 user=postgres password=example dbname=tokendb sslmode=disable

```
