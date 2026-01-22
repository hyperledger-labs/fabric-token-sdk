## The `services/storage` Service

FTS utilizes the following layers of abstraction on top of the database layer:
* `Store`: Executes the SQL queries. A `Store` is only used from within a `StoreService` of the same kind.
* `StoreService`: Extends the `Store` (of the same kind) by adding extra functionality (e.g., keeping maps, cache, or combining functionalities of the underlying store). A `StoreService` does not have any other dependency but the `Store`.
* `Service`: Combines `StoreService` and `Service` instances to provide more complete functionality that can be used by the application.
  Each domain has a `Store`, a `StoreService`, and potentially a `Service`.

The Fabric Token SDK utilizes a robust data management system to ensure the secure and reliable tracking of all token-related activities.
This system leverages several stores, each with a specific purpose.

A single backend system is shared by all databases, offering a more streamlined approach for deployments with simpler requirements.
The specific SQL Driver used by the application will ultimately determine the available deployment options.
For the list of options to configure SQL datasource, refer to the [Fabric Smart Client](https://github.com/hyperledger-labs/fabric-smart-client/) documentation.
Currently, only `SQLite` and `Postgres` are supported.

### Transaction Store (`ttxdb`)

This critical store serves as the central repository for all transaction records.
It captures every token issuance, transfer, or redemption, providing a complete historical record of token activity within the network.
The `ttxdb.StoreService` store is located under `token/services/storage/ttxdb`. It is accessible via the `ttx.Service`.

Here is the data model:

![image](../imgs/ttxdb.png)

Tables:
- `Requests`: Contains the list of `Token Requests` assembled so far. The initial status of a token request is `Pending`.
  When finalized by the backend, it gets status `Valid`. If it is rejected by the backend, it gets status `Invalid`.
  Each token request has a unique identifier.
- `Transactions`: Each Token Request may consist of multiple `Actions`. Each action is represented as a transaction and bound to the same token request.
- `Movements`: Once a transaction is finalized by the backend and its status becomes `Valid`, the transaction gets expanded into its constituent `movements`.
  A movement is either:
    - The transfer of tokens from a source to a destination, or
    - An issuance of tokens to a destination, or
    - The transfer of tokens from a source to void (redemption).
- `Endorsements`: Contains the `endorsement acks` received for a given token request.
- `Validations`: If the node is also a validator (or endorser) of token requests, `Validators` stores the result of the validation of each and every processed token request.

### Token Store (`tokendb`)

The `tokendb` acts as the registry for all tokens within the system.
It stores detailed information about each token, including its unique identifier, denomination type (think currency or unique identifier), current ownership, and total quantity in circulation.
By referencing the `tokendb`, developers and network participants can obtain a clear picture of the token landscape.
The `tokendb.StoreService` is used by the `Token Selector` to select the tokens to use in each transaction, and by the `Token Vault Service` to provide its services.
The `tokendb.StoreService` service is located under `token/services/storage/tokendb`. It is accessible via the `tokens.Service`.

Here are the data models:

![image](../imgs/tokensdb.png)
![image](../imgs/tokenlocksdb.png)

The tables `Tokens` and `TokenOwners` are populated when the backend signals that a given transaction previously assembled is valid.
The table `TokenCertifications` is not used yet.
The table `PublicParameters` is appended with the public parameters as they get announced by the `network service`.

### Audit Transactions Store (`auditdb`)

For applications requiring enhanced auditability, the `auditdb` provides an additional layer of transparency.
It meticulously stores audit records for transactions that have undergone the auditing process.
This functionality is particularly valuable for scenarios where regulatory compliance or tamper-proof records are essential.
The `auditdb.StoreService` is located under `token/services/storage/auditdb`. It is accessible via the `auditor.Service`.
The data model for this DB is identical to that of the transactions DB.

### Identity Store (`identitydb`), Wallet Store (`walletdb`), and Key Store (`keystore`)

As discussed, a Wallet acts like a digital identity vault, holding a `long-term identity` and any `credentials derived from it`.
This identity can take different forms, such as an X.509 Certificate for signing purposes or an [`Idemix Credential`](https://github.com/IBM/idemix) with its associated pseudonyms.

To keep track of the above information we use the following stores:
- `IdentityDB`: Used to store identity configuration, signer related information, audit information, and so on.

  Here is the data model:

  ![image](../imgs/identitiesdb.png)

  The `IdentityConfigurations` table contains the following fields:
    - ID: A unique label to identify this identity configuration.
    - TYPE: Either "Owner", "Issuer", "Auditor", or "Certifier".
      This table stores information about long-term identities upon which wallets are built.

- `WalletDB`: Used to track the mapping between identities, wallet identifiers, and enrollment IDs.

  Here is the data model:

  ![image](../imgs/walletsdb.png)

- `Keystore`: Used for key storage.

  Here is the data model:

  ![image](../imgs/keystore.png)

A wallet must refer to an entry in the `IdentityConfigurations` table via its `id` field.
The wallet identifier is indeed set to the value of the `id` field.
Such an entry contains information about the public version of the `long-term` identity (X.509 Certificate or Idemix Credential).
Secrets are either stored in an external `KeyStore` or spread between `IdentityConfigurations` (raw field) and the Token SDK's `Key Store`.

When an identity is derived from a wallet, its info and signer information are stored in the `IdentityInfo` and `IdentitySigners` tables respectively.
Derivation of an identity involves the generation of secrets that are either stored in the Token SDK Key Store or in an external one.
Depending on the implementation, the generation of these secrets can happen on remote storage.

The `IdentityInfo` table also contains identities of external parties.
Imagine the scenario where Alice transfers tokens to Bob. Bob sends Alice the identity Bob wants to use to receive the tokens, and Alice stores this information in the IdentityInfo table.
In this case, there is no signer associated with this identity; Alice knows then that this is not one of her identities.

