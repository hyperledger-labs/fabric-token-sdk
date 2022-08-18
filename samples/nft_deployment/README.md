# Token SDK, Non Fungible Tokens, Deployment

In this tutorial we will cover the build and deployment process of our [nft sample](../nft).
While the [nft samples](../nft) is focusing on the basic use of the Token SDK, we have disregarded the build and deployment of that application by using the Integration test framework provided by the [Fabric Smart Client](https://github.com/hyperledger-labs/fabric-smart-client).

We recommend completing with the [nft samples](../nft) before continuing here.

TODO define the right order

We will cover the following topics:
- Setup a Fabric network
- Generate crypto material
- Building the Token Validation Chaincode
- Building the FSC node
- Configuration
- Run the node
- Connecting to an existing Fabric network
  - Lazy (use FSC Integration test network)
- Install chaincode
- Run the FSC nodes

## Setup a Fabric network

To illustrate how to use the Fabric Token SDK with an existing network, we will use the XXX/TODO sample network with its default configuration.
We will show the required steps to add Idemix support to the network.

```bash
./network up

# TODO
# create channel
```

### Add Idemix Org



#### Generate crypto material

TODO

To achieve privacy for tokens, the Fabric Token SDK uses Idemix to hide the transaction submitters identities. 
Next we create an Idemix org and add it do our channel.


```bash
go install github.com/IBM/idemix/tools/idemixgen

export IDEMIX_CRYTPO=$(pwd)/testdata/fabric/crypto

idemixgen ca-keygen \
  --output $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com

# issuer
idemixgen signerconfig \
  --ca-input $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com \
  --output $IDEMIX_CRYTPO/peerOrganizations/org1.example.com/peers/issuer.org1.example.com/extraids/idemix \
  --admin \
  -u idemixorg.example.com \
  -e issuer \
  -r 120

# auditor
idemixgen signerconfig \
  --ca-input $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com \
  --output $IDEMIX_CRYTPO/peerOrganizations/org1.example.com/peers/auditor.org1.example.com/extraids/idemix \
  --admin \
  -u idemixorg.example.com \
  -e auditor \
  -r 130

# alice
idemixgen signerconfig \
  --ca-input $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com \
  --output $IDEMIX_CRYTPO/peerOrganizations/org2.example.com/peers/alice.org2.example.com/extraids/idemix \
  --admin \
  -u idemixorg.example.com \
  -e alice \
  -r 140

# bob
idemixgen signerconfig \
  --ca-input $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com \
  --output $IDEMIX_CRYTPO/peerOrganizations/org2.example.com/peers/bob.org2.example.com/extraids/idemix \
  --admin \
  -u idemixorg.example.com \
  -e bob \
  -r 150
```

## FSC Node

The Token SDK builds on top of the Fabric Smart Client (FSC). The business logic implemented using the View API of the Fabric Smart Client is executed by so called FSC nodes. Every participant (i.e., Alice, Bob, the Issuer, and the Auditor) in the nft sample hosts a FSC node. 

### Build

We provide the code for the Alice her FSC node in [nodes/alice/main.go]().
The main function binds the view implementations of the views used by Alice to the FSC node.
For example, Alice uses the `TransferHouseView` 

```go
import (nftsample "github.com/hyperledger-labs/fabric-token-sdk/samples/nft/views")

registry := viewregistry.GetRegistry(n)
if err := registry.RegisterFactory("transfer", &nftsample.TransferHouseViewFactory{}); err != nil {
    return err
}
		
```

### Configuration

Before we can run the FSC node, we need to configure it by providing a `core.yaml`.
We provide a sample configuration file in [nodes/alice/core.yaml]().

The configuration contains three major sections:
```yaml
# this section contains the configuration related to the FSC node
fsc:

# # this section contains the configuration related to the Fabric backend
fabric:

# this section contains the configuration related to the Token SDK
token:
```

### Run

```bash
cd nodes/alice
go build -o alice
./alice node start
```

## Token Validation Chaincode

We begin by building all components used in the nft sample.
The Token SDK uses a Token Validation Chaincode with Fabric to translate Token Transaction into Fabric Transaction with RWSets.
See more details in [TODO])().

### Prepare public parameters

TODO

We are using `tokengen` to create the public parameters used by the Token Validation Chaincode in our nft sample.

### Build

TODO

- Compile TCC and package it using a `Dockerfile`.

### Deploy

Run it as chaincode-as-a-service

TODO

## Action Required: Complete the journey

Next, it's your turn to build, configure, and run the FSC nodes for Bob, the Issuer, and the auditor.
You can use the [toplogy.go](../nft/topology/fabric.go) as a reference point to wire all the views to the nodes. 