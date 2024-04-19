# Services

This section dives into using the Token API to build applications that rely on tokens. 
The Token SDK provides pre-built services specifically designed for token functionality, which we'll explore next. 
But that's not all! The SDK also empowers you to create custom services tailored to your unique needs.

- [`Token Transaction Service`](ttx.md): Simplifies building and managing token transactions across different ledger platforms.
- [`Token Vault Service`](vault.md): Is a secure and adaptable personal vault for managing all your tokens with comprehensive query and retrieval functionalities.
- [`Storage`](storage.md): Fabric Token SDK uses secure databases to track transactions (ttxdb), manage tokens (tokendb), optionally store audit trails (auditdb), and manage user identities (identitydb). 
It offers flexible deployment options for isolated or shared backend systems.
- [`Token Selector`](selector.md): Fabric Token SDK's token selectors allow developers to choose specific tokens (by type, amount, owner) from the vault for transactions. 
They prevent double-spending by locking tokens until the transaction is completed, rejected, times out, or explicitly unlocked
- [`Network`](network.md): Network Service in Fabric Token SDK hides complexities of the ledger (Fabric or Orion) for developers. 
It uses a driver-based design allowing for future support of additional platforms.
- [`Interoperability`](interop.md): Fabric Token SDK allows spending tokens based on conditions defined in scripts. 
You encode the script within the token's owner field, and the backend interprets it during spending. 
This enables interoperability and cross-chain operations.
