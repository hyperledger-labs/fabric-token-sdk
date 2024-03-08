# Storage

The Fabric Token SDK utilizes a robust data management system to ensure the secure and reliable tracking of all token-related activities.
This system leverages several databases, each with a specific purpose:

* **Transaction Database (ttxdb)**:
  This critical database serves as the central repository for all transaction records.
  It captures every token issuance, transfer, or redemption, providing a complete historical record of token activity within the network.
  The ttxdb service is locate under [`token/services/ttxdb`](./../../token/services/ttxdb).

* **Token Database (tokendb)**:
  The tokendb acts as the registry for all tokens within the system.
  It stores detailed information about each token, including its unique identifier, denomination type (think currency or unique identifier), current ownership, and total quantity in circulation.
  By referencing the tokendb, developers and network participants can obtain a clear picture of the token landscape.
  The tokendb is used by the `Token Selector`, to select the tokens to use in each transaction, and by the `Token Vault Service` to provide its services.
  The tokendb service is locate under [`token/services/tokendb`](./../../token/services/tokendb).

* **Audit Database (auditdb)** (if applicable):
  For applications requiring enhanced auditability, the auditdb provides an additional layer of transparency.
  It meticulously stores audit records for transactions that have undergone the auditing process.
  This functionality is particularly valuable for scenarios where regulatory compliance or tamper-proof records are essential. 
  The auditdb service is locate under [`token/services/auditdb`](./../../token/services/auditdb).

* **Identity Database (identitydb)**:
  The identitydb plays a crucial role in managing user identities and wallets within the network.
  It securely stores walet configurations, identity-related audit information, and so on, enabling secure interactions with the token system.
  The identitydb service is locate under [`token/services/identitydb`](./../../token/services/identitydb).

The Fabric Token SDK offers flexibility in deploying these databases. Developers can choose to:

* **Instantiate in Isolation:** Each database can operate independently, utilizing a distinct backend system for optimal performance and manageability.

* **Shared Backend:** Alternatively, a single backend system can be shared by all databases, offering a more streamlined approach for deployments with simpler requirements.

The specific driver used by the application will ultimately determine the available deployment options.
