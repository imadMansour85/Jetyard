# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for a security problem.

Report it privately through GitHub: on this repository, open the **Security** tab, then **Report a vulnerability**. Only the maintainers can read the report.

Please include:

- the version (`GET /version`, or the commit you built from)
- how you deploy it (Docker Compose, Kubernetes, binary) and your NATS server version
- steps to reproduce, and what an attacker gains

## What to expect

- We acknowledge your report within **3 business days**.
- We send a first assessment, and a fix plan if we confirm the issue, within **10 business days**.
- We fix confirmed issues in a release, then publish a GitHub security advisory that credits you, unless you'd rather stay anonymous.

Please give us a reasonable time to ship a fix before disclosing publicly. We'll agree on a date with you.

## Supported versions

Security fixes go into the latest release only. Upgrade to the newest version to get them.

## Scope

How the dashboard stores credentials and authenticates users is documented in [docs/security.md](docs/security.md), including the known limitations. Issues already listed there as limitations are welcome as feature requests, but they are not vulnerabilities.
