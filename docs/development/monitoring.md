# Monitoring

We adopt the monitoring infrastructure provided by the [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/monitoring.md).

We use the following two methods to monitor the performance of the application:
* **Metrics** provide an overview of the overall system performance using aggregated results, e.g. total requests, requests per second, current state of a variable, average duration, percentile of duration
* **Traces** help us analyze single requests by breaking down their lifecycles into smaller components
