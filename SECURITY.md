# Security Policy

## Reporting a Vulnerability

Please use GitHub's private vulnerability reporting feature to report security issues:

**Repository > Security tab > Advisories > Report a vulnerability**

This is the preferred channel. Do not open public issues for security vulnerabilities.

## Scope

The following areas are in scope for vulnerability reports:

- **Kubernetes API access** -- the binary has cluster-wide CRD read permissions
- **Cloudflare API token handling** -- credentials used for Pages deployment
- **Upload logic** -- content published to a public website
- **RBAC configuration** -- permissions granted to the service account
- **Dependency supply chain** -- third-party Go modules and container base images

## Dependency Management

[Renovate](https://docs.renovatebot.com/) automates dependency updates for Go modules, container images, and GitHub Actions to reduce supply chain risk. All GitHub Actions are pinned by commit SHA and Dockerfile base images are pinned by digest.
