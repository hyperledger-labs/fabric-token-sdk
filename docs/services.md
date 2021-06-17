# Services or What you need to build Token-Based Applications

In this Section, we will see how to leverage the Token API to build a token infrastructure on top of Fabric.
This is the most flexible part of the Token SDK stack because new services can be built as needed by the applications.

To build our Fabric token infrastructure, we will need the following building blocks (or services).

- On the Fabric-side, we will define a `Token Namespace` that will contain all information that is needed to make 
  sense of the tokens stored in the namespace. The namespace's endorsement policy depends on the specific use case.
  As a general guideline, we can say it should reflect the majority of the partners in the blockchain consortium.
  Last but not least, a chiancode, the `Token Chaincode`. It is deployed to create the token namespace.
  The token chaincode provides various functionalities to the applications.
  We will explore these functions in more details later.
- On the Client-side, we need
    - A `Token Transaction` struct that let parties agree on the token operations to perform 
      (issue, transfer, redeem, and so on).
    - A `Token Vault`. A local storage that contains tokens. We will learn exactly which tokens the vault contains
      in the coming sections.
    - A `Token Selector` to pick tokens from the vault and use them in operations like transfer and redeem.
    - `Auditing` to enforce rules on a token transaction before this gets committed. This part is optional.
  
Let us start with the token namespace and its chaincode.

## The Token Namespace and Chaincode

The `Token Namespace` contains the tokens and any additional information needed to make sense out of them
in a key-value format. Attached to the namespace we have: 
- The [`Token Chaincode`](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/services/tcc),
  that exposes functionalities useful to develop token applications, and 
- The `Endorsement Policy` that describe the governance of the token chaincode. In other words, who is allowed to
modify the namespace.
  
The Token Chaincode must be initialized with some `public parameters`. 
The public parameters depends on the specific token driver implementation.
Once initialized, the token chaincode can provide the following functionalities:
- `Fetch the public parameters`. They must be fetched by each FSC node running the Token SDK stack. 
  This is done automatically. 
- `Register issuers and auditors`. Indeed, only certain parties can issue tokens and audit token operations.
- `Fetch Tokens` is used to retrieve the content of tokens by their ids.  
- `Validate and Translate Token Requests`. This is one of the essential steps in the lifecycle of a token transaction.
In the next Section, we will understand this function better. 

## Token Transaction Lifecycle

The lifecycle of a Token Transaction consists of the following high-level steps:
1. `Assembling the Token Transaction`. In this phase, the business parties decide on the token operations
that must happen atomically. They assemble them in a token transaction by interacting following a business process.
   Actually, the Token Transaction contains a Token Request that we have seen being the container of Token Actions.
2. `Collect Signatures`. Once the Token Transaction is ready, one of the business parties takes the charge of
collecting the following signatures:
   - From the issuers of new tokens;
   - From the owners of the tokens spent;
   - From any required auditor, if needed.

3. `Collect Endorsements`. The endorsers of the Token Chaincode that must validate the Token Transaction;

4. `Submit the Token Transaction for Ordering`. At this stage the token transaction can be submitted to the ordering
service to apply the changes to the Token Namespace.
   