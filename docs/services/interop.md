# Spendable Scripts for Cross-Chain Swaps

The Token SDK lets you create tokens that can be programmed! These "spendable scripts" allow for secure cross-chain operations.

Here's how it works:

* **Script as Owner:**  Instead of a regular owner, you can encode a script as the owner of a token. Different drivers in the system can understand and execute these scripts.
* **Script Evaluation:** When you spend a token with a script as its owner, the script is evaluated at that moment. The script determines if the spending is allowed based on its programmed conditions.

## Secure Swaps with HTLC

One powerful use case for spendable scripts is Hash Time Locked Contracts (HTLC). This lets you securely swap tokens between different blockchains.

* **Locking the Tokens:**  To initiate a swap, you lock your token in a script. This script defines the recipient, a deadline, and a secret code (hashlock).
* **Recipient Claims the Token:** The recipient can only claim the token by providing the secret code (preimage) before the deadline.
* **Timelock Protection:** If the recipient doesn't claim the token on time, the script will allow you to get the token back.

## Script Details

The HTLC script stores all the necessary information:

* Identities of both sender and recipient
* Deadline for claiming the token
* Hash information (including the hash itself, the hashing function used, and the encoding)

Here's a glimpse of the code structure for the script and hash information:

```go
// Script details for HTLC
type Script struct {
    Sender    string // Identity of the sender
    Recipient string // Identity of the recipient
    Deadline  time.Time
    HashInfo  HashInfo
}

// Information about the hashlock
type HashInfo struct {
    Hash         []byte
    HashFunc     string // Hashing function used (e.g., SHA-256)
    HashEncoding string // Encoding format (e.g., Base64)
}
```

## Handy Services for Script Management

The Token SDK provides several services, under [`token/services/interop`](./../../token/services/interop), to manage scripts and HTLC swaps:

* **Building Transactions:**  A service helps you build transactions with actions like locking (initiating a swap), claiming (recipient receiving the token), and reclaiming (sender getting the token back if unclaimed).
* **Wallet Interactions:**  A separate wallet service lets you list tokens with specific preimages or find expired tokens (where the deadline has passed).
* **Script-Specific Services:**  Additional services handle signing messages (including the preimage for HTLC) and verifying script ownership.
* **Driver Integration:**  Existing drivers like FabToken and ZKAT DLog are already compatible with interoperability and HTLC functionality. These drivers have enhanced validation rules to ensure proper script execution and deadline adherence.


For a deeper dive into specific drivers, refer to the FabToken and ZKAT DLog documentation.
