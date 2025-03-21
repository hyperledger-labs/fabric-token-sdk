# Zero Knowledge Asset Transfer DLog Driver

The `Zero Knowledge Asset Transfer DLog` (zkat-dlog, for short) driver supports privacy using Zero Knowledge Proofs. 
We follow a simplified version of the blueprint described in the paper <!-- markdown-link-check-disable -->
[`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')<!-- markdown-link-check-disable -->
by Elli Androulaki, Jan Camenisch, Angelo De Caro, Maria Dubovitskaya, Kaoutar Elkhiyaoui, and Björn Tackmann.
In more details, the driver hides the token's owner, type, and quantity.
But it reveals which token has been spent by a give transaction. We say that this driver does not support `graph hiding`.
Owner anonymity and unlinkability is achieved by using Identity Mixer (Idemix, for short).
The identities of the issuers and the auditors are not hidden.

The above scheme is secure under `computational assumptions in bilinear groups` in the `random-oracle model`.

The driver implementation is available under the folder [`nogh`](./../../token/core/zkatdlog/nogh).

## Key characteristic

- A token is represented on the ledger as the pair `(pedersen commitment to type and value, owner)`.
- The owner of a token can be:
  - An `Idemix Identity`;
  - An `HTLC-like Script`;
  - A `Multisig Script`.
- An issuer is identified by an X509 certificate. The identity of the issuer is always revealed.
- An auditor is identified by an X509 certificate. The identity of the auditor is always revealed.
- Supported actions are: `Issue` and `Transfer`. `Reedem` is obtained as a `Transfer` that creates an output whose's owner is `none`.
- An `Issue Action` proves ...
- A Transfer Action proves ...
- All the information required to operate the driver are found in the public parameters.