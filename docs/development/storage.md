# Storage

For an introduction into the concepts of Database, Persistence, Driver, Store, read [this documentation](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/platform/view/db-driver.md).

The project utilizes the following layers of abstraction on top of the database layer:
* `Store`: executes the SQL queries. A `Store` is only used from within the `StoreService` of the same kind.
* `StoreService`: extends the `Store` (of the same kind) by adding extra functionality (e.g. keeping maps, cache, or combining functionalities of the underlying store). A `StoreService` does not have any other dependency, but the `Store`.
* `Service`: combines `StoreService` and `Service` instances to provide more complete functionality that can be used by the application.
  Each domain has a `Store`, a `StoreService` and potentially a `Service`.

The Fabric Token SDK utilizes a robust data management system to ensure the secure and reliable tracking of all token-related activities.
This system leverages several stores, each with a specific purpose:

* **Transaction Store (`ttxdb`)**:
  This critical store serves as the central repository for all transaction records.
  It captures every token issuance, transfer, or redemption, providing a complete historical record of token activity within the network.
  The `ttxdb.StoreService` store is located under [`token/services/ttxdb`](./../../token/services/ttxdb). It is accessible via the `ttx.Service`.

* **Token Store (`tokendb`)**:
  The `tokendb` acts as the registry for all tokens within the system.
  It stores detailed information about each token, including its unique identifier, denomination type (think currency or unique identifier), current ownership, and total quantity in circulation.
  By referencing the `tokendb`, developers and network participants can obtain a clear picture of the token landscape.
  The `tokendb.StoreService` is used by the `Token Selector`, to select the tokens to use in each transaction, and by the `Token Vault Service` to provide its services.
  The `tokendb.StoreService` service is located under [`token/services/tokendb`](./../../token/services/tokendb). It is accessible via the `tokens.Service`.

* **Audit Store (`auditdb`)** (if applicable):
  For applications requiring enhanced auditability, the `auditdb` provides an additional layer of transparency.
  It meticulously stores audit records for transactions that have undergone the auditing process.
  This functionality is particularly valuable for scenarios where regulatory compliance or tamper-proof records are essential.
  The `auditdb.StoreService` is located under [`token/services/auditdb`](./../../token/services/auditdb). It is accessible via the `auditor.Service`.

* **Identity Store (`identitydb`) and Wallet Store (`walletdb`)**:
  The `identitydb` plays a crucial role in managing user identities and wallets within the network.
  It securely stores wallet configurations, identity-related audit information, and so on, enabling secure interactions with the token system.
  The `identitydb.StoreService` is located under [`token/services/identitydb`](./../../token/services/identitydb).

## Configuration

The Token SDK offers flexibility in deploying these databases. Developers can choose to:

* **Instantiate in Isolation:** Each database can operate independently, utilizing a distinct backend system for optimal performance and manageability.
  Here is an example of configuration
```yaml
token:
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, etc)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable
      # db specific driver
      tokendb:
        persistence: my_token_persistence
```

* **Shared Backend:** Alternatively, a single backend system can be shared by all databases, offering a more streamlined approach for deployments with simpler requirements.
```yaml
token:
  tms:
    mytms: # unique name of this token management system
      network: default # the name of the network this TMS refers to (Fabric, etc)
      channel: testchannel # the name of the network's channel this TMS refers to, if applicable
      namespace: tns # the name of the channel's namespace this TMS refers to, if applicable
```

The specific driver used by the application will ultimately determine the available deployment options.
Don't forget to import the driver that you are ultimately using with a blank import in your executable.

For the list of options to configure sql datasources, refer to the [Fabric Smart Client documentation](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/core-fabric.md).