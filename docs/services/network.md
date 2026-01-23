## The Network Service

The `network` service, located under `token/services/network`, provides other services with a consistent and predictable interface to the backend (e.g., Fabric).
Internally, the network service mirrors the structure of the Token API, consisting of a `Provider` of network instances and the `Network` instances themselves.

The network service architecture is depicted below:

![network_service.png](../imgs/network_service.png)

The Fabric-based network implementation utilizes the Fabric Smart Client for configuration and operations, including chaincode queries and transaction broadcasting.

During bootstrap, the Token SDK processes the TMS defined in the configuration.
For each TMS, the network provider retrieves the network instance corresponding to the `network` and `channel` specified in the TMS ID.
Failure to retrieve the network instance results in a bootstrap failure.
Upon success, the `Connect` function on the `Network` instance is invoked with the target namespace.
This function establishes a connection to the backend, enabling the Token SDK to receive updates on public parameters and transaction finality.

When `Connect` is called, the Fabric network implementation establishes two `Fabric Delivery` streams to receive committed blocks:
- One stream is used to analyze transactions that update the public parameters.
  More specifically, for each transaction in a block, the parser checks if the RW set contains a write whose key is the `setup key`.
  The setup key is set as `\x00seU+0000`. If such a key is found in a valid transaction, then the listener added upon calling `Connect` does the following:
    - It invokes the `Update` function of the TMS provider passing the TMS ID and the byte representation of the new public parameters.
      The function works as follows: if a TMS instance with the passed ID does not exist, it creates one.
      If a TMS with that ID already exists, then:
        - A new instance of TMS is created with the new public parameters.
        - If the previous step succeeds, then the `Done` function on the old TMS instance is invoked to release all allocated resources.
    - If the above step succeeds, then the public parameters are appended to the `PublicParameters` table.
- The other stream is dedicated to transaction finality. Services can add listeners to the `Network` instance to listen for the finality of specific transactions.
  The `ttx` service and the `audit` service add a listener when a transaction has reached the point of being ready to be submitted to the ordering service.
  (For more information, look at the sections dedicated to these services). Both services use the same listener.
  This listener performs the following actions upon notification of the finality of a transaction:
    - If the transaction's status is valid, then the token request's hash contained in the transaction is matched against the hash of the token request stored in the database.
      If they match, then the `Tokens` table is updated by inserting the new tokens and marking the spent tokens as deleted.
      The corresponding token request in the `Requests` table is marked as `Valid` with a change of the status field.
    - If the transaction's status is invalid, then the corresponding token request in the `Requests` table is marked as `Invalid` or `Deleted`.
      In all other cases, an error is returned.

