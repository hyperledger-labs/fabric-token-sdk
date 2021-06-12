## Examples via Integration Tests

Integration tests are useful to show how multiple components work together.
The Fabric Smart Client comes equipped with some of them to show the main features.
To run the integration tests, you need to have Docker installed and ready to be used.

Each integration test bootstraps the FSC and Fabric networks as needed, and initiate the
business processes by invoking the `initiator view` on the specific FSC nodes.

Here is a list of available examples:

- [`Tha Basics`](./token/tcc/basic/README.md): A showcase of all possibility that the Token SDK offers.
- [`I Owe You`](./token/dvp/README.md): In this example, we see how to orchestrate a Delivery vs Payment use-case

