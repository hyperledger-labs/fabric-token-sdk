# Services

Services provide pre-built functionalities designed to streamline the utilization of the Token API. 
Certain services, such as the `Identity Service`, also serve as foundational components for `Drivers`.

The interaction between `Services` is illustrated below:

![services.png](imgs/services.png)

Key components include:
*   `config service`: Manages the Token SDK configuration.
*   `identity service`: Handles identity management, including wallets, long-term identities (X.509 and Idemix), and associated stores. It supports portions of the Token and Driver APIs.
*   [`ttx service`](./services/ttx.md): The **Token Transaction Service**. Facilitates the assembly of token requests and transactions for the backend. It is backend-agnostic, relying on the `network service` for backend-specific operations.
*   [`network service`](./services/network.md): The **Network Service**. Provides a unified interface for interacting with diverse networks or backends (e.g., Fabric), abstracting implementation details from other services.
*   [`storage service`](./services/storage.md): The **Storage Service**. Encapsulates storage mechanisms required by the Token SDK, supporting the Token and Driver APIs.
*   `selector service`: The **Token Selector Service**. mitigating the risk of double-spending by implementing strategic token selection algorithms.

