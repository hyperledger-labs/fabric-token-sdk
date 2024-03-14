# Identity Service: Who Can Do What in the Token-SDK?

The Token-SDK uses identities to establish trust and control access within the system. 
These identities are like digital passports that verify who a party is and what actions they're authorized to perform.

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
A wallet is like a digital vault that stores a long-term identity (the main key) and any credentials derived from it. 
Different wallet types provide different functionalities. 
For example, an issuer wallet lets you see a list of issued tokens, while an owner wallet shows unspent tokens.

To manage these identities and wallets, the identity service provides several tools:

* **Identity Provider:** Manages roles and the identities within them.
* **Wallet Registry:** Uses the Identity Provider to manage all wallets associated with a specific role. 
This registry relies on the token driver to implement the specific wallet functionalities defined by the Driver API.

In essence, the Token-SDK identity service provides a secure and flexible framework for managing access control within your system.

The identity service is locate under [`token/services/identity`](./../../token/services/identity).

## Understanding Roles in More Detail

Building on the concept of long-term identities, we'll now explore how they are grouped into roles within the identity service.

Each role acts as a container for long-term identities, which are then used to create wallets. Here's the interface that defines a role:

```go
// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role interface {
	// ID returns the identifier of this role
	ID() driver.IdentityRole
	// MapToID returns the long-term identity and its identifier for the given index.
	// The index can be an identity or a label (string).
	MapToID(v interface{}) (view.Identity, string, error)
	// GetIdentityInfo returns the long-term identity info associated to the passed id
	GetIdentityInfo(id string) (driver.IdentityInfo, error)
	// RegisterIdentity registers the given identity
	RegisterIdentity(id string, path string) error
	// IdentityIDs returns the identifiers contained in this role
	IdentityIDs() ([]string, error)
	// Reload the roles with the respect to the passed public parameters
	Reload(pp driver.PublicParameters) error
}
```

This interface offers functions for managing identities within the role. 
You, as the developer, have the flexibility to implement a role using any identity representation that best fits your application's needs. 
For example, a role could even encompass identities based on various cryptographic schemes.

The identity service conveniently provides two built-in implementations of the Role interface. 
Both implementations leverage the concept of Hyperledger Fabric MSP ([https://hyperledger-fabric.readthedocs.io/en/latest/msp.html](https://hyperledger-fabric.readthedocs.io/en/latest/msp.html)):

* [**MSP X.509:**](./../../token/services/identity/msp/x509) This implementation retrieves long-term identities from local folders adhering to the X.509-based MSP format.
* [**MSP Idemix:**](./../../token/services/identity/msp/idemix) This implementation loads long-term identities from local folders that follow the Idemix-based MSP format.

## Using the Identity Service in a Token Driver

If you want to use the Identity Service in your Token Driver, then here is what you need to do.

First, instantiate your roles. The identity service come equipped with two `Role` implementation.
One is based on X.509 certificates, the other one is based on Idemix. 
Both requires the identities to be stored on the filesystem following the Hyperledger Fabric MSP prescriptions. 

High level, these are the steps to follow:
1. Instantiate the roles your driver will support.
2. Instantiate the Wallet Registries for each role.

Let see an example taken from the [`fabtoken`](./../../token/core/fabtoken/driver) driver.