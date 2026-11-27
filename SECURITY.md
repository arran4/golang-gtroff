# Security

Security fixes are applied to the development branch, `main`. There is no stable release support policy yet.

## Report a vulnerability

Email [arran4@gmail.com](mailto:arran4@gmail.com). Use GitHub private vulnerability reporting if enabled. Include the affected version or commit, steps to reproduce, and the impact. Please avoid public disclosure of exploitable details or credentials until coordinated.

## Security boundaries

Parsing and rendering do not invoke a system groff interpreter. The optional evaluator interprets a limited roff subset, not arbitrary shell commands. Source-set loading resolves bounded local includes; AST inspection does not load includes. Read, memory and recursion limits reduce resource-exhaustion risk but do not constitute an OS sandbox.

The HTTP library serves a read-only snapshot of explicit roots, defaults to loopback, and rejects traversal and untrusted Host headers. The CLI's optional system search and desktop viewer may invoke explicitly selected host programs. See [source loading](docs/sources.md), [evaluation](docs/evaluation.md) and [HTTP library](docs/serve.md) for details.
