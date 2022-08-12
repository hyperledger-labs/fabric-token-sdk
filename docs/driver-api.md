# Driver API

The `Driver API`, located in `token/driver` package, defines the contracts any driver should respect to be compatible with the Token API.
It has a finer granularity than the Token API to better accommodate the differences among different token technologies.

The first interface to implement is the `driver.Driver` interface. 
The Driver interface is used to create instances of the Token Manager Service, the Public Parameters Manager, and Validator. 
Then a driver must be registered in order to be used by the Token Stack.
This is usually done by calling the static function `Register` from the  `token/core` package.

Here is an example that leverage the golang `package initialization` mechanism to register the driver:
```go
func init() {
	core.Register("my-driver", &Driver{})
}
```
To enable the driver, it is enough to have a blank import of the driver package.

The registered drivers are used by the `core` package to create instances of the Token Manager Service, the Public Parameters Manager, and Validator
for different networks. The `core.TMSProvider` takes care of that.

Let us have a look at the other interfaces that the driver must implement:

## Token Management Service

The `Token Management Service` interface (`driver.TokenManagerService`) (TMS, for short) is the entry point of the Driver API.

## Public Params Manager

The `Public Params Manager` interface (`driver.PublicParamsManager`) is used to manage the public parameters of the token.

## Token Request

The `Token Request` struct (`driver.TokenRequest`) is the struct used by the Driver API to model the token request.
It contains the serialized version of the actions and the witnesses. 

## Issue Service

The `Issue Service` interface (`driver.IssueService`) contains the API to generate an instance of the `driver.IssueAction` interface. 
This Issue Action can then be serialized and appended to a token request. 

## Transfer Service

The `Transfer Service` interface (`driver.TransferService`) contains the API to generate an instance of `driver.TransferAction` interface.
This Transfer Action can then be serialized and appended to a token request.

## Token Service

The `Token Service` interface (`driver.TokenService`) contains the token-related API.
For example, the developer can use this interface to unmarshal a serialized output using metadata to derive a token and its issuer (if any). 

## Auditor Service

The `Auditor Service` interface (`driver.AuditorService`) contains the Auditor API.
An auditor can use this API to verify the well-formedness of a token request with the respect to given metadata and anchor.

## Wallet Service

The `Wallet Service` interface (`driver.WalletService`) contains the API to manage all the wallets.

## Identity Provider

The Identity Provider interface (`driver.IdentityProvider`) is used to manage long-term identities on top of which wallets are defined.
The provider handles also signature verifiers, signers, and identity audit info.

Each identity must have a unique role: Issuer, Auditor, Owner or Certifier.

[//]: # ( 
TODO:
Should we also explain somewhere what these roles are?
By reading this Issuer, Auditor and Owner are clear to me but what is the extra role of the Certifier? Maybe others reader will have similar thoughts?
)

## Validator

The Validator interface (`driver.Validator`) is used to validate a token request.

## Config Manager

The Config Manager interface (`config.Manager`) is used to manage the configuration of the driver.