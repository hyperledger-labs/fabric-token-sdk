# Token SDK, Non Fungible Tokens, Deployment

In this tutorial we will cover the build and deployment process of our [nft sample](../nft).
While the [nft samples](../nft) is focusing on the basic use of the Token SDK, we have disregarded the build and deployment of that application by using the Integration test framework provided by the [Fabric Smart Client](https://github.com/hyperledger-labs/fabric-smart-client).

We recommend completing with the [nft samples](../nft) before continuing here.

TODO define the right order

We will cover the following topics:
- Setup a Fabric network
- Generate crypto material
- Building the FSC node
- Configuration
- Run the node
- Connecting to an existing Fabric network
  - Lazy (use FSC Integration test network)
- Building the Token Chaincode
- Install & deploy Token Chaincode
- Run the FSC nodes

## Setup a Fabric network

To illustrate how to use the Fabric Token SDK with an existing network, we will use microFab (TBD?!?!?)) as our Fabric network with the following configuration:
1. Two organization: Org1 and Org2;
2. Single channel;
3. Org1 runs/endorses the Token Chaincode.

We also need to add Idemix support to the network to allow business parties to hide their identities when submitting Fabric transactions.

To start the network run:
```bash
just microfab
```

### Add Idemix Org

We need to add an Idemix organization to allow business parties to sign Fabric transactions in an anonymous and unlikable way.

#### Generate crypto material

First step is to generate the Idemix Credential Issuer's key pair:

```bash
go install github.com/IBM/idemix/tools/idemixgen

export IDEMIX_CRYTPO=$(pwd)/testdata/fabric/crypto

idemixgen ca-keygen \
  --output $IDEMIX_CRYTPO/peerOrganizations/idemixorg.example.com

```

Then, we need to generate the Idemix Credential Signer's key pairs for Alice and Bob, our business parties:

```bash
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

TODO 

### Add Idemix Org to channel

1. Add idemixOrgs to network https://hyperledger-fabric.readthedocs.io/en/latest/idemix.html
2. Update Channel configuration


## Token Validation Chaincode

We begin by building all components used in the nft sample.
The Token SDK uses a Token Validation Chaincode with Fabric to translate Token Transaction into Fabric Transaction with RWSets.
See more details in [TODO])().

### Generate crypto material

TODO: We need to generate crypto material for the Issuer, the Auditor, and the Token Owners.
Issuer and Auditor are X509 certificates. For Token Owners, we need to setup an Idemix CA with curve `BN254`. 

### Prepare public parameters

We are using `tokengen` to create the public parameters used by the Token Validation Chaincode in our nft sample.

tokengen gen fabtoken

```bash
go install github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen
tokengen gen dlog \ 
  --auditors $IDEMIX_CRYTPO/peerOrganizations/org1.example.com/peers/auditor.org1.example/com/extraid/idemix/msp \ 
  --issuers $IDEMIX_CRYTPO/peerOrganizations/org1.example.com/peers/issuer.org1.example/com/extraid/idemix/msp\
  --cc

cat zkatdlog_pp.json | base64
```

Copy the base64-encoded params into `$FTS_PATH/token/services/network/tcc/params.go`

```go
package tcc

const Params = `>>>BASE64_STRING<<<`
```


### Build

Once you have updated the `params.go`, you can compile TCC and package it using a `Dockerfile`.

```bash
go build $FTS_PATH/token/services/network/tcc/main
```

TODO

### Deploy

Run it as chaincode-as-a-service

TODO

## FSC Node

The Token SDK builds on top of the Fabric Smart Client (FSC). The business logic implemented using the View API of the Fabric Smart Client is executed by so called FSC nodes. Every participant (i.e., Alice, Bob, the Issuer, and the Auditor) in the nft sample hosts a FSC node. 

### Create crypto material

TODO add

- FSC node crypto material
  - p2p
  - grpc
  - tls
  - wallets

#### FTS Wallet crypto material (idemix)

```bash

go install github.com/hyperledger-labs/fabric-smart-client/cmd/fsccli

fsccli cryptogen generate \
  --config $TOKEN_CRYPTO/default-testchannel-zkat/issuer/issuers/crypto-config.yaml \
  --output $TOKEN_CRYPTO/default-testchannel-zkat/issuer/issuers

fsccli cryptogen generate \
  --config $TOKEN_CRYPTO/default-testchannel-zkat/auditor/auditors/crypto-config.yaml \
  --output $TOKEN_CRYPTO/default-testchannel-zkat/auditor/auditors
 
idemixgen signerconfig \
  --ca-input $TOKEN_CRYPTO/default-testchannel-zkat/idemix \
  --output $TOKEN_CRYPTO/default-testchannel-zkat/idemix/alice \
  --admin \
  -u default-testchannel-zkat.example.com \
  -e alice \
  -r 100 \
  --curve BN254 

idemixgen signerconfig \
  --ca-input $TOKEN_CRYPTO/default-testchannel-zkat/idemix \
  --output $TOKEN_CRYPTO/default-testchannel-zkat/idemix/bob \
  --admin \
  -u default-testchannel-zkat.example.com \
  -e bob \
  -r 110 \
  --curve BN254 

```

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
  id: fsc.alice
  networkId: 7jxtlwbgxfachim4clepdnqu7q
  # grpc view manager endpoint
  address: 127.0.0.1:20006
  addressAutoDetect: true
  listenAddress: 127.0.0.1:20006 # TODO is this redundant with address?
  
  # FSC node identity for p2p comm
  identity:
    cert:
      file: $FSC_CRYPTO/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/msp/signcerts/alice.fsc.example.com-cert.pem
    key:
      file: $FSC_CRYPTO/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/msp/keystore/priv_sk
  admin: # what is that used for?
    certs:
      - /Users/bur/Developer/gocode/src/github.com/hyperledger-labs/fabric-token-sdk/samples/nft/testdata/fsc/crypto/peerOrganizations/fsc.example.com/users/Admin@fsc.example.com/msp/signcerts/Admin@fsc.example.com-cert.pem
  tls:
    # this is for grpc-tls connection
  web:
  tracing:
  metrics:
  endpoint:
    resolvers:
    # resolvers used for p2p authentication
    - name: issuer
      domain: fsc.example.com
      identity:
        id: issuer
        path: $FSC_CRYPTO/peerOrganizations/fsc.example.com/peers/issuer.fsc.example.com/msp/signcerts/issuer.fsc.example.com-cert.pem
      addresses:
        P2P: 127.0.0.1:20001
      aliases:

# this section contains the configuration related to the fabric driver 
fabric:
  default:
    mspConfigPath: $CRYTPO/peerOrganizations/org2.example.com/peers/alice.org2.example.com/msp
    msps:
      - id: idemix
        mspType: idemix
        mspID: IdemixOrgMSP
        cacheSize: 0
        path: $CRYTPO/peerOrganizations/org2.example.com/peers/alice.org2.example.com/extraids/idemix
    tls:
    peers:
    channel:
    vault:
    endpint:
      resolvers:
        # can't we use discovery here?

# this section contains the configuration related to the Token SDK
token:
  enabled: true
  tms:
    - certification: null
      channel: testchannel
      namespace: zkat
      network: default
      wallets:
        owners:
          - default: true
            id: alice
            path: $TOKEN_CRYPTO/default-testchannel-zkat/idemix/alice
  ttxdb:
    persistence:
      opts:
        path: $WD/testdata/fsc/nodes/alice/kvs
      type: badger
```

### Run

```bash
cd nodes/alice
go build -o alice
./alice node start
```

## Action Required: Complete the journey

Next, it's your turn to build, configure, and run the FSC nodes for Bob, the Issuer, and the auditor.
You can use the [toplogy.go](../nft/topology/fabric.go) as a reference point to wire all the views to the nodes. 