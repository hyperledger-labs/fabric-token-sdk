# Linting

golangci-lint is a highly efficient and widely used command-line tool for linting Go source code. 
It acts as a runner for multiple independent linters, executing them in parallel to speed up the analysis process. 
By aggregating the results from various linters, golangci-lint helps identify a broad range of potential issues, 
including code style violations, programming errors, complexity problems, and potential security vulnerabilities, 
contributing to cleaner and more maintainable Go projects.

To install the linter, follow the instruction here: [`https://golangci-lint.run/welcome/install/#local-installation`](https://golangci-lint.run/welcome/install/#local-installation).

To run the linter for the project, just run

```bash
make lint
```

To change the configuration of the linter, check the [`.golangci.yml`](./../../.golangci.yml).
This configuration is also used by our CI.
