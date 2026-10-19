# Security Policy

## Supported Versions

The current `main` branch is supported. Older versions are best-effort.

## Reporting a Vulnerability

**Do not create a public issue.** Email security@<your-domain>. You will receive:

- a brief acknowledgement within 72 hours;
- a severity assessment and fix timeline;
- credit in the release notes (if you want it).

Please include:

- description and proof-of-concept;
- affected version / commit;
- expected impact.

## What counts as a vulnerability

- XSS, SQLi, SSRF, RCE in the main code or templates.
- CSRF bypass (`http.CrossOriginProtection`).
- Auth / session bypass.
- Hardcoded secrets, dependency leakage.

Out of scope: browser issues, denial of service with no economic gain for the attacker,
theoretical attack vectors without a PoC.

Thanks for responsible disclosure.
