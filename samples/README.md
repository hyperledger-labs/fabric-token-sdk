# Samples

Samples are a collection of small and simple apps that demonstrate how to use the library.

To run the samples, we recommend to use `go 1.18`. You will also need docker when using Fabric.
To make sure you have all the required docker images, you can run `make docker-images` in the
folder `$GOPATH/src/github.com/hyperledger-labs/fabric-token-sdk`.

- [`Fungible Tokens, The Basics. On Fabric`](./fungible//README.md): How to handle `fungible tokens`.
- [`Non-Fungible Tokens, The Basics. On Fabric`](./nft//README.md): How to handle `non-fungible tokens`.

## Additional Examples via Integration Tests

Integration tests are useful to show how multiple components work together.
The Fabric Smart Client comes equipped with some of them to show the main features.
To run the integration tests, you need to have Docker installed and ready to be used.

Each integration test bootstraps the FSC and Fabric networks as needed, and initiate the
business processes by invoking the `initiator view` on the specific FSC nodes.

All integration tests can be executed by executing `make integration-tests`
from the folder `$GOPATH/github.com/hyperledger-labs/fabric-token-sdk`.
Each test can be executed either using your preferred IDE or by executing `go test` from
the folder that contains the test you want to try.

Here is a list of available examples:

- [`DvP`](../integration/token/dvp/README.md): In this example, we see how to orchestrate a Delivery vs Payment use-case

## Further information

Almost all the samples and integration tests require the fabric binaries to be downloaded and the environment variable `FAB_BINS` set to point to the directory where these binaries are stored. One way to ensure this is to execute the following in the root of the fabric-smart-client project

```shell
make download-fabric
export FAB_BINS=$PWD/../fabric/bin
```
