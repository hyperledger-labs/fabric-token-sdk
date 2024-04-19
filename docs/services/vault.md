# Token Vault Service

Located within the [`token/services/vault`](./../../token/services/vault) package, the Token Vault service acts as your personal vault for all the tokens you hold.
It meticulously tracks every token currently residing in the wallets under your control, regardless of whether they were directly issued to you or transferred from external parties.

One of the key strengths of the Token Vault service is its adaptability.
It leverages the network service to seamlessly connect with the specific ledger backend's local vault instance.
This translates to a technology-agnostic solution that functions flawlessly regardless of the underlying infrastructure.

The Token Vault service empowers you with a comprehensive suite of functionalities for managing your tokens:

* **Query Service:** This service provides insightful queries to help you navigate your token holdings effectively.
  Here's a glimpse into some of the valuable functionalities offered:

    * **IsPending:** This function clarifies the status of a transaction by returning `true` if the transaction with the provided ID remains pending and `false` if it has been completed.
    * **IsMine:** As the name suggests, this function verifies ownership. It returns `true` if the provided token ID belongs to any of your known wallets.
    * **Unspent Token Iterators:** These iterators act as powerful tools for exploring your unspent tokens. They come in two flavors:
        * **UnspentTokensIterator()**: This iterator efficiently traverses through all your unspent tokens.
        * **UnspentTokensIteratorBy(id, typ string)**: This specialized iterator allows you to filter your unspent tokens based on ownership and type. You can specify a wallet ID (`id`) and an optional token type (`typ`). If no type is provided, the iterator returns tokens of any type owned by the specified wallet.
    * **List Functions:** The Token Vault service offers several list functions to provide consolidated views of your tokens:
        * **ListUnspentTokens()**: This function returns a comprehensive list of all your unspent tokens.
        * **ListAuditTokens(ids ...*token.ID)**: This function retrieves a list of audited tokens associated with the provided token IDs.
        * **ListHistoryIssuedTokens()**: This function delivers a detailed list of all tokens that have been issued within the network.
    * **Public Parameters:** The PublicParams() function retrieves the public parameters associated with the token system.
    * **Token Information Retrieval:** These functions provide mechanisms to access information about your tokens:
        * **GetTokenInfos(ids []*token.ID)**: This function retrieves information for the provided token IDs.
        * **GetTokenOutputs(ids []*token.ID)**: Similar to the previous function, this function retrieves the raw token outputs stored on the ledger for the provided IDs.
        * **GetTokenInfoAndOutputs(ids []*token.ID)**: This function offers a combined approach, retrieving both the token information and their corresponding outputs for the provided IDs.
    * **GetTokens:** This function retrieves a list of tokens along with their corresponding vault keys.
    * **WhoDeletedTokens:** This function delves into the history of deleted tokens. It provides information about who deleted the specified tokens (if applicable) and returns a boolean array indicating whether each token at a given position has been deleted.

* **Certification Service (if applicable):**
  This service provides functionalities for managing token certifications, which are additional layers of validation used in certain token systems.

The Token Vault service equips you with a robust and adaptable toolkit for managing your tokens.
Its rich set of functionalities empowers you to maintain a clear and secure grasp on your token holdings.
