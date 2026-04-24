# OpenSSF Best Practices Assessment: Fabric Token SDK
**Baseline Level 3 (Silver) (v2025.10.10)**

This document provides a self-assessment of the Fabric Token SDK project against the OpenSSF Best Practices Baseline Level 3 (Silver) criteria.

| ID | Criterion / Question | Assessment | Notes |
| :--- | :--- | :--- | :--- |
| **Controls** | | | |
| OSPS-AC-04.02 | Minimum privileges assigned in CI/CD pipeline? | **Met** | Workflow files (`golangci-lint.yml`, `codeql-analysis.yml`) explicitly define restricted `permissions`. |
| OSPS-BR-02.02 | Release assets clearly associated with release identifier? | **Met** | Releases are tagged via Git and managed as Go Modules, following SemVer. |
| OSPS-BR-07.02 | Policy for managing secrets and credentials? | **Met** | Project mandates protecting secrets (documented in `AGENTS.md`) and uses GitHub Secrets. |
| OSPS-DO-03.01 | Instructions to verify integrity of release assets? | **Not Met** | No explicit documentation found for verifying checksums or GPG signatures of releases. |
| OSPS-DO-03.02 | Instructions to verify identity of release author? | **Not Met** | No explicit documentation for verifying release authorship. |
| OSPS-DO-04.01 | Descriptive statement about scope/duration of support? | **Not Met** | While versioning is documented, a formal support lifecycle/duration statement is missing. |
| OSPS-DO-05.01 | Descriptive statement on EOL for security updates? | **Not Met** | No explicit EOL policy for specific versions documented. |
| OSPS-GV-04.01 | Policy for reviewing code collaborators prior to escalation? | **Met** | Standard Hyperledger/LFDT practice; maintainers are listed in `MAINTAINERS.md`. |
| OSPS-QA-02.02 | Release assets delivered with a SBOM? | **Not Met** | No Software Bill of Materials (SBOM) generation currently implemented. |
| OSPS-QA-04.02 | Subprojects enforce strict security requirements? | **Met** | All core logic is co-located and follows the same CI/CD and security standards. |
| OSPS-QA-06.02 | Documentation clearly documents when and how tests run? | **Met** | Detailed in `docs/development/testing.md` and `Makefile`. |
| OSPS-QA-06.03 | Policy that major changes add/update automated tests? | **Met** | Mandated in project contribution guidelines (`AGENTS.md`). |
| OSPS-QA-07.01 | Require at least one non-author approval before merge? | **Met** | "One Approve Policy" documented in `DEVELOPMENT.md`. |
| OSPS-SA-03.02 | Perform threat modeling and attack surface analysis? | **Not Met** | No formal threat modeling document is publicly available in the repository. |
| OSPS-VM-04.02 | Vulnerabilities not affecting project documented in VEX? | **Not Met** | No Vulnerability Exploitability eXchange (VEX) documents found. |
| OSPS-VM-05.01 | Policy defines threshold for remediation of SCA findings? | **Met** | Zero-tolerance policy for linting/security findings implemented in CI. |
| OSPS-VM-05.02 | Policy to address SCA violations prior to release? | **Met** | Standard PR gate requirement for all Hyperledger projects. |
| OSPS-VM-05.03 | Automatic evaluation and blocking for malicious dependencies?| **Met** | GitHub/CodeQL dependency scanning enabled. |
| OSPS-VM-06.01 | Policy defines threshold for remediation of SAST findings? | **Met** | CI gates fail on CodeQL/SAST findings. |
| OSPS-VM-06.02 | Automatic evaluation and blocking for security weaknesses? | **Met** | Implemented via CodeQL and `golangci-lint` in GitHub Actions. |
