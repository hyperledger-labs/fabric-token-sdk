# Integration tests

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
