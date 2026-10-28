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

You should receive an acknowledgment within 48 hours, followed by status updates as the issue is investigated and remediated.

## Pre-Publication Secret Auditing & Guidelines

The repository undergoes automated and manual secret auditing prior to any public visibility transitions:
- Scanned with pinned secret detection tools ([Gitleaks](https://github.com/gitleaks/gitleaks) and [TruffleHog](https://github.com/trufflesecurity/trufflehog)) covering all reachable Git history, tags, and pull request references.
- GitHub Actions logs, artifacts, and issue/PR discussions are audited for accidental credentials.
- No real tokens, private keys, cloud credentials, or sensitive network endpoints are permitted in the codebase or test fixtures.

### Contributor Responsibilities
- Never commit genuine API keys, personal access tokens, private keys, passwords, or internal infrastructure hostnames.
- For test cases requiring credential-like syntax or certificates, use RFC 2606 / RFC 6761 reserved domains (`example.com`, `example.invalid`) and clearly labeled mock/synthetic data.
- Ensure temporary files, test outputs, and debugging logs remain ignored via `.gitignore`.

## Security Boundaries & Resource Limits

`golang-gtroff` implements defensive bounds against malformed or hostile input:
- **Path Traversal & Host Isolation**: The HTTP library (`internal/serve`, `gtroff serve`) rejects directory traversal sequences (`..`), forbidden control characters, and untrusted `Host` headers.
- **Resource Limits**: Configurable read budgets (`MaxFileBytes`, `MaxMemoryBytes`) and recursion depth limits protect against memory exhaustion or infinite macro expansion loops.
- **Pure-Go Sandboxing**: Headless CLI binaries require no CGO and operate without executing external shell processes for core operations.
