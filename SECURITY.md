# Security Policy

## Supported Versions

`golang-gtroff` is currently an active work-in-progress groff/troff parser and toolchain written in pure Go. Security updates and bug fixes are applied to the `main` branch.

| Version / Branch | Supported          | Status |
| ---------------- | ------------------ | ------ |
| `main`           | :white_check_mark: | Active development |
| Prior branches   | :x:                | Superseded |

## Reporting a Vulnerability

If you discover a security vulnerability or sensitive information disclosure in `golang-gtroff`, please report it responsibly rather than opening a public issue.

- **Email**: Send details to [arran4@gmail.com](mailto:arran4@gmail.com).
- **GitHub**: Use GitHub's private vulnerability reporting feature once enabled on the repository settings.

Please include:
- A description of the issue and potential security impact.
- Steps to reproduce, test fixtures, or proof of concept.
- Affected commands, packages, or HTTP endpoints (e.g., `gtroff serve`, parser bounds, path handling).

The maintainer will aim to acknowledge reports promptly and provide updates as the issue is investigated. This is a best-effort project without a guaranteed response-time SLA.

## Public-history and release auditing

Issue [#75](https://github.com/arran4/golang-gtroff/issues/75) tracks the public-history and artifact audit. A policy describing intended scans is not evidence that a past visibility transition was cleared. The [dated audit checkpoint](docs/security-audit/README.md) records actual inputs, pinned tools, redacted findings and remaining access/retention limits. License and source-provenance review remains separate in [#67](https://github.com/arran4/golang-gtroff/issues/67).

Before approving a release candidate or another visibility transition:

- Enumerate all accessible branch, tag and PR refs in a fresh full-history clone, and record their SHAs. Include accessible deleted-branch history retained by PR refs.
- Run pinned [Gitleaks](https://github.com/gitleaks/gitleaks) and [TruffleHog](https://github.com/trufflesecurity/trufflehog), distinguish findings from scan failures, and classify false positives without hiding them.
- Inspect actual retained GitHub Actions logs/artifacts and issue/PR discussions. State which original artifacts are no longer available; rebuilding an equivalent candidate does not audit the old bytes.
- Review applicable GitHub security/protection settings without inferring their state from an inaccessible API or an empty alert list. Record maintainer actions separately.
- No real tokens, private keys, cloud credentials, or sensitive network endpoints are permitted in the codebase or test fixtures.

Keep potentially sensitive inputs restricted. Never publish raw scanner matches, credential values, diagnostic bodies or private exports. The optional Go audit helper emits only candidate location/rule metadata and scan status; review that metadata before publishing it. A completed scan is not publication clearance or proof that every possible exposure was detected. If a real credential exposure is confirmed, revocation/rotation is required; rewriting Git history alone does not revoke it.

### Contributor Responsibilities
- Never commit genuine API keys, personal access tokens, private keys, passwords, or internal infrastructure hostnames.
- For test cases requiring credential-like syntax or certificates, use RFC 2606 / RFC 6761 reserved domains (`example.com`, `example.invalid`) and clearly labeled mock/synthetic data.
- Ensure temporary files, test outputs, and debugging logs remain ignored via `.gitignore`.

## Security Boundaries & Resource Limits

`golang-gtroff` implements defensive bounds against malformed or hostile input:
- **Path Traversal & Host Isolation**: The HTTP library (`manlib`, `gtroff serve`) rejects directory traversal sequences (`..`), forbidden control characters, and untrusted `Host` headers.
- **Resource Limits**: Configurable read budgets (`MaxFileBytes`, `MaxMemoryBytes`) and recursion depth limits protect against memory exhaustion or infinite macro expansion loops.
- **Headless implementation**: Core parsing and rendering paths are pure Go and do not invoke an external groff interpreter. This does not constitute an operating-system security sandbox.
