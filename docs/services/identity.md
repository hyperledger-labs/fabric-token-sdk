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
The role is defined by the [`Role`](./../../token/services/identity/roles.go) interface.
This interface offers functions for managing identities within the role. 
You, as the developer, have the flexibility to implement a role using any identity representation that best fits your application's needs. 

A default implementation is provided under [`token/services/identity/role`](./../../token/services/identity/role).

## Understanding Wallets in More Detail

The Token-SDK abstracts the wallet management via an interface called [`WalletService`](./../../token/driver/wallet.go) that is part of the `Driver API`.
The `WalletService` interface gives access to the available wallets.
You, as the developer, have the flexibility to implement a `WalletService` that best fits your application's needs.

A default implementation is provided under [`token/services/identity/wallet`](./../../token/services/identity/wallet).

## Storage

The identity service uses 3 data storage defined by the following interfaces:
- `IdentityDB`: It is used to store identity configuration, signer related information, audit information, and so on.
- `WalletDB`: It is used to track the mapping between identities, wallet identifier, and enrollment IDs.
- `Keystore`: It is used for the key storage.

### Implementation

We support the following implementations:
- `IdentityDB`, can be either based on the `identitydb` service or the Fabric-Smart-Client's KVS.
- `WalletDB`, same as the `IdentityDB`.
- `Keystore` is based on the Fabric-Smart-Client's KVS.
By default, the `identitydb` is used to provide both an implementation to both the `IdentityDB` and `WalletDB`.

To retrieve the implementation of these interfaces, we have the `identity.StorageProvider` interface.
An implementation for this interface can be found under [`token/sdk/identity`](../../token/sdk/support/identity).
It uses the `identitydb` service for the `IdentityDB` and the `WalletDB`, and the Fabric-Smart-Client's KVS for the `Keystore`.

### HashiCorp Vault Secrets Engine Support

The HashiCorp Vault Secrets Engine is a modular component of Vault designed to securely manage, store, or generate sensitive data such as API keys, passwords, certificates, and encryption keys.
The identity service provides an implementation for both the `IdentityDB` and `Keystore` based on the `HashiCorp Vault Secrets Engine`.
This implementation can be found under [`hashicorp`](./../../token/services/identity/storage/kvs/hashicorp).
This implementation requires to configure the `HashiCorp Vault Secrets Engine` to run in non-versioned mode (i.e., stores the most recently written value for a key). 
For more information about non-versioned secrets engine mode, refer to the (https://developer.hashicorp.com/vault/docs/secrets/kv/kv-v1).

In order to use this integration, the developer must do the following:
1. Implement the `identity.StorageProvider` interface. 
2. Register this implementation in [`Dig`](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/sdk.md) via decoration like this:
   ```go
    p.Container().Decorate(NewMixedStorageProvider)
   ```
   This can be added in the Application Dig SDK that links that token-sdk Dig SDK.

Here is an example of the implementation of the `identity.StorageProvider` interface implementing bullet `1`:

```go
type MixedStorageProvider struct {
	kvs     kvs.KVS
	manager *identitydb.Manager
}

func NewMixedStorageProvider(client *vault.Client, prefix string, manager *identitydb.Manager) (*MixedStorageProvider, error) {
	kvs, err := hashicorp.NewWithClient(client, prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating hashicorp.NewWithClient")
	}
	return &MixedStorageProvider{kvs: kvs, manager: manager}, nil
}

func (s *MixedStorageProvider) WalletDB(tmsID token.TMSID) (identity.WalletDB, error) {
	return s.manager.WalletServiceByTMSId(tmsID)
}

func (s *MixedStorageProvider) IdentityDB(tmsID token.TMSID) (identity.IdentityDB, error) {
	return kvs.NewIdentityDB(s.kvs, tmsID), nil
}

func (s *MixedStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
```

 