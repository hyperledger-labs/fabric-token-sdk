# Auditor

The `auditor` service provides auditing capabilities for token transactions. 
The service interacts with an audit database and a network provider to track and manage transaction status.

Key features and components:

- **Auditor:** The central service responsible for auditing transactions.
    - Uses the `auditdb` service to store audit records.
    - Relies on a `NetworkProvider` to interact with networks and channels.
- **Auditing flow:**
  1. **Validate**: Checks the validity of a token request using `request.AuditCheck()`.
    2. **Audit**: Extracts inputs and outputs from a transaction, locking enrollment IDs for safety.
    3. **Append**: Adds a transaction to the audit database and subscribes to transaction status changes on the network.
    4. **Release**: Releases locks acquired during auditing.
- **Querying and status management:**
    - **NewPaymentsFilter**: Creates a PaymentFilter to query movements from the database
    - **NewHoldingsFilter**: Creates a HoldingsFilter to query holdings from the database
    - **SetStatus**: Sets the status of an audit record (Pending, Confirmed, Deleted).
    - **GetStatus**: Retrieves the status of a transaction.
    - **GetTokenRequest**: Retrieves the token request associated with a transaction ID.

The auditor service is located under [`token/services/auditor`](./../../token/services/auditor).