# OpenSSF Best Practices Assessment: Fabric Token SDK
**Baseline Level 1 (v2025.10.10)**

This document provides a self-assessment of the Fabric Token SDK project against the OpenSSF Best Practices Baseline Level 1 criteria.

| ID | Criterion / Question | Assessment | Notes |
| :--- | :--- | :--- | :--- |
| **1. Basics** | | | |
| - | Human-readable name and brief description? | **Met** | Fabric Token SDK: APIs and services for token-based applications. |
| - | Project and version control URLs? | **Met** | https://github.com/hyperledger-labs/fabric-token-sdk |
| - | License(s) in SPDX format? | **Met** | Apache-2.0 |
| - | Programming language(s) used? | **Met** | Go (1.24+) |
| **2. Access Control** | | | |
| OSPS-AC-01.01 | Require MFA for sensitive resources? | **Met** | GitHub/Hyperledger maintainer policy requires MFA. |
| OSPS-AC-02.01 | New collaborators restricted to lowest privileges? | **Met** | Standard fork/PR workflow; access granted by maintainers. |
| OSPS-AC-03.01 | Enforcement against direct commits to primary branch? | **Met** | Branch protection enabled on `main`. |
| OSPS-AC-03.02 | Require confirmation before deleting primary branch? | **Met** | GitHub standard safety mechanism. |
| **3. Build & Release** | | | |
| OSPS-BR-01.01 | CI/CD pipeline input parameters sanitized/validated? | **Met** | Standard GitHub Actions isolation and parameter handling. |
| OSPS-BR-01.02 | CI/CD branch names sanitized/validated? | **Met** | Handled by GitHub Actions triggers. |
| OSPS-BR-03.01 | Official project channels delivered via HTTPS/SSH? | **Met** | All Git and web traffic is encrypted. |
| OSPS-BR-03.02 | Distribution channels delivered via HTTPS/SSH? | **Met** | Binary/tool downloads in Makefile use HTTPS. |
| OSPS-BR-07.01 | Prevent unencrypted secrets in VCS? | **Met** | Uses `.gitignore`, GitHub Secrets, and CodeQL scanning. |
| **4. Documentation** | | | |
| OSPS-DO-01.01 | User guides for basic functionality? | **Met** | Extensive documentation in `docs/` and `README.md`. |
| OSPS-DO-02.01 | Guide for reporting defects? | **Met** | Issues handled via GitHub; instructions in `CONTRIBUTING.md`. |
| OSPS-GV-02.01 | Mechanisms for public discussion? | **Met** | Discord channel and GitHub Issues/PRs. |
| OSPS-GV-03.01 | Contribution process explained? | **Met** | Detailed in `CONTRIBUTING.md`. |
| OSPS-VM-02.01 | Documentation contains security contacts? | **Met** | Provided in `SECURITY.md` (security@hyperledger.org). |
| **5. Licensing & Quality**| | | |
| OSPS-LE-02.01 | Source code uses OSI/FSF-approved license? | **Met** | Apache-2.0. |
| OSPS-LE-02.02 | Released assets use OSI/FSF-approved license? | **Met** | Covered under the project license. |
| OSPS-LE-03.01 | License file included in the repository? | **Met** | `LICENSE` file present in root. |
| OSPS-LE-03.02 | License file included alongside release assets? | **Met** | Included in repository and distribution packages. |
| OSPS-QA-01.01 | Source code publicly readable at a static URL? | **Met** | GitHub repository is public. |
| OSPS-QA-01.02 | Full record of changes maintained? | **Met** | Complete Git history available. |
| OSPS-QA-02.01 | Repository contains a dependency list? | **Met** | `go.mod` and `go.sum` are present. |
| OSPS-QA-04.01 | All subprojects listed in documentation? | **Met** | All core components (`token`, `integration`, etc.) are documented. |
| OSPS-QA-05.01 | VCS excludes generated executable artifacts? | **Met** | Excluded via `.gitignore`. |
| OSPS-QA-05.02 | VCS excludes unreviewable binary artifacts? | **Met** | Only source and config are tracked. |
