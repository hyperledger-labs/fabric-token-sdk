# Token Transaction Service

The Token SDK simplifies token transaction assembly with a dedicated service in the [`token/services/ttx`](./../../token/services/ttx) package.
This service handles the entire transaction lifecycle, from building the transaction to managing its various stages.

Even better, this service is ledger-agnostic.
Whether you're using Fabric, Orion, or another platform, you can use the same service to assemble token transactions.
This flexibility is thanks to the `token/services/network` service, which acts as an abstraction layer, hiding the complexities of the underlying ledger technology.
We'll delve deeper into this service later.

Now, let's break down the high-level steps involved in a token transaction lifecycle:

1. **Assemble the Token Transaction:**
   During this phase, the involved parties collaborate to define the token operations that need to occur atomically (all at once).
   They achieve this by assembling these operations into a token transaction, following an interactive business process.
   Behind the scenes, a token transaction includes a Token Request, as specified by the Token API.

2. **Collect Endorsements:**
   Once the token transaction is ready (meaning the Token Request is finalized), one party, designated as the leader, takes charge of the following steps:

    - **Gather Signatures:** The leader collects signatures (endorsements) from relevant parties for each action:
        - Issuers of any new tokens (if applicable)
        - Owners of any tokens being spent (if applicable)
    - **Request Audit:**
      The leader sends the token transaction to an auditor for verification. If all checks pass, the auditor signs the transaction and returns the signature to the leader.
      This step is optional
    - **Request Approval:** Now, the transaction needs to be validated and converted into a format compatible with the ledger backend. The leader strips all private data from the transaction and sends it to approvers for validation and translation. These approvers send back the translated transaction signed with their approvals. The leader then attaches these approvals to the original transaction.
    - **Distribute Approvals:** Finally, the leader distributes the complete token transaction, including endorsements, to all participating parties.

3. **Commit:** With everything in place, the transaction is ready to be committed. The leader sends the transaction to the ledger backend (e.g., the ordering service in Fabric), again removing any private information. The leader and all other parties can then wait for confirmation (finality) from the ledger backend, indicating that the transaction is committed to the local vault.
