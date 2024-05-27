# Storage

The Fabric Token SDK utilizes a robust data management system to ensure the secure and reliable tracking of all token-related activities.
This system leverages several databases, each with a specific purpose:

* **Transaction Database (`ttxdb`)**:
  This critical database serves as the central repository for all transaction records.
  It captures every token issuance, transfer, or redemption, providing a complete historical record of token activity within the network.
  The `ttxdb` service is locate under [`token/services/ttxdb`](./../../token/services/ttxdb).

* **Token Database (`tokendb`)**:
  The `tokendb` acts as the registry for all tokens within the system.
  It stores detailed information about each token, including its unique identifier, denomination type (think currency or unique identifier), current ownership, and total quantity in circulation.
  By referencing the `tokendb`, developers and network participants can obtain a clear picture of the token landscape.
  The `tokendb` is used by the `Token Selector`, to select the tokens to use in each transaction, and by the `Token Vault Service` to provide its services.
  The `tokendb` service is locate under [`token/services/tokendb`](./../../token/services/tokendb).

* **Audit Database (`auditdb`)** (if applicable):
  For applications requiring enhanced auditability, the `auditdb` provides an additional layer of transparency.
  It meticulously stores audit records for transactions that have undergone the auditing process.
  This functionality is particularly valuable for scenarios where regulatory compliance or tamper-proof records are essential. 
  The `auditdb` service is locate under [`token/services/auditdb`](./../../token/services/auditdb).

* **Identity Database (`identitydb`)**:
  The `identitydb` plays a crucial role in managing user identities and wallets within the network.
  It securely stores wallet configurations, identity-related audit information, and so on, enabling secure interactions with the token system.
  The `identitydb` service is locate under [`token/services/identitydb`](./../../token/services/identitydb).

## Configuration

The Token SDK offers flexibility in deploying these databases. Developers can choose to:

* **Instantiate in Isolation:** Each database can operate independently, utilizing a distinct backend system for optimal performance and manageability.
Here is an example of configuration 
```yaml
token:
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, Orion, etc)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable
      # db specific driver
      tokendb:
        persistence:
          type: sql
          opts:
            createSchema: true 
            driver: sqlite    
            maxOpenConns: 10
            dataSource: /some/path/tokendb
```

* **Shared Backend:** Alternatively, a single backend system can be shared by all databases, offering a more streamlined approach for deployments with simpler requirements.
```yaml
token:
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, Orion, etc)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable

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
```

The specific driver used by the application will ultimately determine the available deployment options.
Don't forget to import the driver that you are ultimately using with a blank import in your executable.  
