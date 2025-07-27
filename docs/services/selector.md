# Token Selector

The Fabric Token SDK offers a powerful concept known as token selectors, empowering developers with granular control over token selection within the vault.
These selectors function as filters, allowing you to specify precise criteria for choosing the tokens you require for a particular transaction.

Here's a breakdown of how token selectors work:

* **Conditional Selection:** You can define a set of conditions to narrow down the pool of available tokens.
  Common examples include selecting tokens based on:
    * **Type:** Specify the desired token denomination (e.g., "USD Coin" or a unique identifier token).
    * **Amount:** Define the exact quantity of tokens required for the transaction.
    * **Ownership:** Select tokens held within a specific wallet.

* **Preventing Double Spending:** To safeguard against the potential for double-spending (using the same token in multiple transactions), token selectors enforce a locking mechanism.
  Once a token is selected, it becomes temporarily unavailable for other transactions until its fate is determined.
  This locking remains in place until one of the following scenarios occurs:
    * **Transaction Commitment:** If the transaction using the selected tokens is successfully committed to the backend, the lock is released.
    * **Transaction Rejection:** If the transaction is rejected, the lock is lifted, and the tokens become available for selection again.
    * **Timeout:** If a predefined period of inactivity elapses (timeout), the lock automatically expires, and the tokens are released.
    * **Explicit Unlock:** Developers can also choose to explicitly unlock tokens before the transaction is completed.

By leveraging token selectors, developers can ensure they are working with the appropriate tokens for their transactions while maintaining the integrity of the system and preventing fraudulent activities like double-spending.

We currently support two selector types:
- `Simple`: It is mostly used for testing.  
- [`Sherdlock`](./selector/sherdlock.md): This is a fast selector that works well in replication settings too.

The selector service is locate under [`token/services/selector`](../../token/services/tokens/selector).