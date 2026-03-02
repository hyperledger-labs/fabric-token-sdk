# OpenSSF Best Practices Assessment: Fabric Token SDK
**Baseline Level 2 (v2025.10.10)**

This document provides a self-assessment of the Fabric Token SDK project against the OpenSSF Best Practices Baseline Level 2 criteria.

| ID | Criterion / Question | Assessment | Notes |
| :--- | :--- | :--- | :--- |
| **1. Security Architecture & Design** | | | |
| OSPS-SA-01.01 | Design documentation for actions and actors? | **Met** | Detailed in `docs/driverapi.md`, `docs/services.md`, and `docs/tokensdk.md` with PUML diagrams. |
| OSPS-SA-02.01 | Documentation of external software interfaces (APIs)? | **Met** | Public APIs documented in `docs/tokenapi.md` and `token/driver`. |
| OSPS-SA-03.01 | Security assessment performed? | **?** | The `README.md` states the project "has not been audited". While CodeQL and security guidelines exist, a formal threat model document is not explicitly linked. |
| **2. Vulnerability Management** | | | |
| OSPS-VM-01.01 | Coordinated Vulnerability Disclosure (CVD) policy? | **Met** | Standard Hyperledger policy in `SECURITY.md` and linked Defect Response page. |
| OSPS-VM-03.01 | Private reporting of vulnerabilities? | **Met** | Private email (security@hyperledger.org) provided in `SECURITY.md`. |
| OSPS-VM-04.01 | Public disclosure of vulnerabilities? | **Met** | Hyperledger publishes security bulletins for confirmed issues. |
| **3. Access Control & Governance** | | | |
| OSPS-AC-04.01 | CI/CD tasks default to lowest possible permissions? | **Met** | Workflow files (`golangci-lint.yml`, `codeql-analysis.yml`) explicitly define restricted `permissions`. |
| OSPS-GV-01.01 | Documentation lists project members with sensitive access? | **Met** | Maintainers are listed in `MAINTAINERS.md`. |
| OSPS-GV-01.02 | Description of roles and responsibilities? | **Met** | Roles (Maintainer vs Emeritus) are listed in `MAINTAINERS.md`; process is governed by the LFDT Charter. |
| OSPS-GV-03.02 | Contributor guide outlines requirements? | **Met** | Detailed requirements (DCO, rebase, coding standards) in `CONTRIBUTING.md`. |
| OSPS-LE-01.01 | Contributors assert legal authority (DCO/CLA)? | **Met** | Mandatory DCO sign-off (`-s`) for every commit is enforced. |
| **4. Build, Release & Quality** | | | |
| OSPS-BR-02.01 | Every official release has a unique version? | **Met** | Follows SemVer; practices documented in `docs/development/versioning.md`. |
| OSPS-BR-04.01 | Releases include changelogs (functional/security)? | **Met** | Standard Hyperledger release practice via GitHub Releases and changelogs. |
| OSPS-BR-05.01 | Standardized tooling for dependencies? | **Met** | Uses Go Modules (`go.mod`). |
| OSPS-BR-06.01 | Signed releases or signed manifests? | **Met** | Hyperledger releases typically include signed SHASUMs or GPG-signed tags. |
| OSPS-DO-06.01 | Documentation describes dependency policy? | **Met** | `go.mod` for tracking; `CONTRIBUTING.md` and `AGENTS.md` mention dependency practices. |
| OSPS-QA-03.01 | Automated status checks for primary branch? | **Met** | GitHub Actions (`tests.yml`, `golangci-lint.yml`) must pass for all PRs. |
| OSPS-QA-06.01 | Automated test suite runs in CI/CD? | **Met** | Comprehensive unit and integration tests run on every PR. |
