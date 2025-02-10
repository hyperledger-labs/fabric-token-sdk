# Identity Service: Who Can Do What in the Token-SDK?

The Token-SDK uses identities to establish trust and control access within the system. 
These identities are like digital passports that verify who a party is and what actions they're authorized to perform.
Therefore, the identity service must offer:
1. A way to define these identities, 
2. A way to generate and verify digital signatures valid under these identities, and
3. A way to marshal/unmarshal identities and signatures.

**Think of Roles as Teams:**

Within the identity service, long-term identities are grouped into roles. 
Imagine these roles as teams with specific permissions. 
Here are some key roles:

* **Issuers:** Like a mint that creates coins, issuers have the power to create new tokens.
* **Owners:** Owners hold tokens, just like possessing a digital asset.
* **Auditors:** These act as financial inspectors, overseeing token requests and ensuring proper use.
* **Certifiers:** They verify the existence and legitimacy of specific tokens, similar to checking identification.
This role is used only by certain token drivers that support the so called `graph hiding`.

**Identity Options: Passports and Beyond**

Identities can come in different forms. 
A common choice is an X.509 certificate, a secure electronic passport that links a real-world entity (website, organization, or person) to a public key using a digital signature. 
This certificate uniquely identifies the entity it represents.

Another option is anonymous credentials, which allow users to prove they have certain attributes (like age or qualifications) without revealing their entire identity. 
Imagine showing an ID that only displays relevant information, not your full details. 
This is particularly useful for protecting user privacy.

**Roles and Wallets: Managing Access**

Importantly, roles can contain different identity types. 
These roles then act as the foundation for creating wallets. 
A wallet is like a digital vault that stores a long-term identity (the main key) and any credentials derived from it, if supported. 
Different wallet types provide different functionalities. 
For example, an issuer wallet lets you see a list of issued tokens, while an owner wallet shows unspent tokens.
All these wallets have though a minimum common denominator: 
Given a wallet, you, as the developer, can derive identities to be used in different contexts.
For example, given an owner wallet, you can derive the identities that will own tokens. 
To spend these tokens, the wallet will give access to the signers bound to these identities.
The signers can be used to generate the signatures necessary to spend these tokens.

In essence, the Token-SDK identity service provides a secure and flexible framework for managing access control within your system.

The identity service is locate under [`token/services/identity`](./../../token/services/identity).

## Understanding Roles in More Detail

Building on the concept of long-term identities, we'll now explore how they are grouped into roles within the identity service.

Each role acts as a container for long-term identities, which are then used to create wallets. 
Here's the interface that defines a role:

```go
// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role interface {
	// ID returns the identifier of this role
	ID() IdentityRoleType
	// MapToIdentity returns the long-term identity and its identifier for the given index.
	// The index can be an identity or a label (string).
	MapToIdentity(v WalletLookupID) (Identity, string, error)
	// GetIdentityInfo returns the long-term identity info associated to the passed id
	GetIdentityInfo(id string) (IdentityInfo, error)
	// RegisterIdentity registers the given identity
	RegisterIdentity(config IdentityConfiguration) error
	// IdentityIDs returns the identifiers contained in this role
	IdentityIDs() ([]string, error)
}
```

This interface offers functions for managing identities within the role. 
You, as the developer, have the flexibility to implement a role using any identity representation that best fits your application's needs. 

A default implementation is provided under [`token/services/identity/role`](./../../token/services/identity/role).

## Understanding Wallets in More Detail

The Token-SDK abstracts the wallet management via a service called `WalletService`. 
Here is the interface that defines such a service:

```go
// WalletService models the wallet service that handles issuer, recipient, auditor and certifier wallets
type WalletService interface {
	// RegisterRecipientIdentity registers the passed recipient identity together with the associated audit information
	RegisterRecipientIdentity(data *RecipientData) error

	// GetAuditInfo retrieves the audit information for the passed identity
	GetAuditInfo(id Identity) ([]byte, error)

	// GetEnrollmentID extracts the enrollment id from the passed audit information
	GetEnrollmentID(identity Identity, auditInfo []byte) (string, error)

	// GetRevocationHandle extracts the revocation handler from the passed audit information
	GetRevocationHandle(identity Identity, auditInfo []byte) (string, error)

	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(identity Identity, auditInfo []byte) (string, string, error)

	// Wallet returns the wallet bound to the passed identity, if any is available
	Wallet(identity Identity) Wallet

	// RegisterOwnerIdentity registers an owner long-term identity
	RegisterOwnerIdentity(config IdentityConfiguration) error

	// RegisterIssuerIdentity registers an issuer long-term wallet
	RegisterIssuerIdentity(config IdentityConfiguration) error

	// OwnerWalletIDs returns the list of owner wallet identifiers
	OwnerWalletIDs() ([]string, error)

	// OwnerWallet returns an instance of the OwnerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	OwnerWallet(id WalletLookupID) (OwnerWallet, error)

	// IssuerWallet returns an instance of the IssuerWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	IssuerWallet(id WalletLookupID) (IssuerWallet, error)

	// AuditorWallet returns an instance of the AuditorWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	AuditorWallet(id WalletLookupID) (AuditorWallet, error)

	// CertifierWallet returns an instance of the CertifierWallet interface bound to the passed id.
	// The id can be: the wallet identifier or a unique id of a view identity belonging to the wallet.
	CertifierWallet(id WalletLookupID) (CertifierWallet, error)

    // SpendIDs returns the spend ids for the passed token ids
    SpendIDs(ids ...*token.ID) ([]string, error)
}
```

The `WalletService` gives access to the available wallets.
You, as the developer, have the flexibility to implement a `WalletService` that best fits your application's needs.

A default implementation is provided under [`token/services/identity/wallet`](./../../token/services/identity/wallet).