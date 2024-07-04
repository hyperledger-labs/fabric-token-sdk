# Sherdlock token selector

## Motivation

### Problems with simple and mailman selector

The current token-selector implementation does not support node replication, as it assumes that only one FSC node accesses the tokens of a specific wallet/currency combination. This leads to the following problems:

* When one replica adds a token, another replica is not aware of this. Workaround for mailman: Fetch tokens from the DB when the unspent tokens iterator finishes.
* When one replica spends a token, another replica still tries to use the token. This will cause a double-spending error. Workaround for mailman: We can repeat by refreshing the unspent tokens iterator in case of a duplicate error.
* When one replica adds a token for a different wallet/currency combination, the manager returns an error that there is no mailman for this combination. Workaround for mailman: If we can't find a wallet/currency combination, fetch all tokens from the DB.
* Each time two replicas try to spend the same token, only the first one will succeed and the second has to repeat the token selection from scratch.

### Possible workaround

A possible solution would be to assign each user to a specific replica (sticky sessions). However:

* this could create uneven traffic to some replicas.
* replicas are not idempotent, so failure of a replica would require that another replica be assigned to each user or adding a replica would mean that a redistribution of the users should be done.
* sticky sessions should be used as an optimisation and not part of the implementation logic.

## Solution

### Requirements

A desired solution would:

* serve any token-selection request for any wallet/currency
* support addition/removal of replicas seamlessly
* keep up with tokens locked/added/deleted by other replicas in real time so it avoids to lock unavailable tokens
* not deteriorate read/write operations on the token table for other consumers
* eventually unlock tokens in case of failure

### Implementation and Algorithm

For this, we can use a shared lock table with the following fields:

* `tokenID` The token we want to lock
* `spenderTxID` The transaction that has locked the token
* `createdAt` The timestamp when the token was locked


Each replica keeps a cache (queue) of tokens. At the beginning this queue may be empty or half filled by a previous selection. We note down that more tokens may be available if we query the database (but we do not query yet, as we may already have enough tokens in our cache).

1. Pop an element from the token queue.
   1. If the cache is not empty, try to lock the token.
      1. If the token was locked by another process, then note that there may be still available tokens if we query the database. (The other process may release it eventually).
      2. If we successfully locked the token, add the token value to the sum and return the result, if the desired quantity was achieved.
   2. If the cache is empty (no token was found):
      1. If we had not noted that other tokens may be available, we do not need to query the database. There are no other tokens and we have not collected the desired amount, so we abort.
      2.    If other tokens may be available, we can retry to fetch the tokens from the token DB and populate our cache. If we have exceeded a maxRetries limit, then we abort, as this might lead to deadlocks. Otherwise with the newly-populated cache, we start from step 1.

Notes:

* Whenever we abort, we release the acquired locks (remove the entries from the lock DB).
* When a replica collects the desired amount, it will eventually spend (delete) the tokens from the DB, but the token-lock entries in the lock DB will not be removed yet. This helps other replicas to see that this token is locked, so they cannot lock it and they pass to the next available token. When their token iterator finishes, they will fetch the tokens from the DB and will not try to lock again the spent token.
* Suppose we have 2 tokens of value CHF3 each (CHF6 in total) and 2 replicas that try to spend CHF4. Then each replica might lock one token each and then retry (maxRetries times) until the other replica unlocks the tokens. After that the process will abort ideally for one of the two, but for a specific timing, it might abort for both (livelock). As an enhancement, we can backoff a randomly selected backoff interval within a range and retry. If the backoff interval for one replica is sufficiently shorter than for the other, then the first replica will acquire both locks. If the backoff intervals are very close, then we will end up with the same situation and retry with another backoff. However, if a node has left the tokens locked because it crashed, we will need to backoff and retry again and again, until these locks are considered expired and cleaned up by a housekeeping job (see below).

### Further optimisations / Future work

* We can subscribe to the token DB write operations and populate the token cache whenever a new token is added/deleted.
* A housekeeping job can remove all "expired" locks (There createdAt field is greater than e.g. 1 minute ago). These could be locks that correspond to:
* tokens that have been spent and hence deleted (the other replicas will only try at most once to lock them, so it doesn't increase the complexity).
tokens that have been locked by a replica that crashed.
* When a replica tries to lock a token, it might try to lock a token that it has already locked. Instead of trying to insert optimistically in the DB and getting an error, it could first make a local check.
* If we have multiple replicas iterating over the same tokens in the same order, they will also try to lock them in the same order and that will create higher contention. A permutation can be applied with every query in the token DB.
* Sticky sessions (a specific replica serves a specific user wallet/currency combination) can eliminate contention between different replicas.