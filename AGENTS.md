# Agent guidance

## Parser and token contracts
- This package parses groff/troff input, especially man and ms macros, into typed tokens and extracts text. Preserve documented ParseLine, token and macro contracts; do not claim byte-for-byte round trips unless tests and APIs explicitly promise them.
- Differentiate preserved source representation, semantic tokens and any intentionally canonical output. Do not silently discard unsupported lines or macros if the public API promises visibility or error reporting.
- Keep parser/test cases small and readable: requests, quoted/escaped arguments, missing parameters, malformed input, multiline material and unknown macros. Verify error behaviour separately from valid tokenization.
- Keep generated macro tables or other generated Go files tied to their actual source templates and regeneration process; inspect README's development section first.

## Agent follow-through
Run Go tests and the documented generation check (where applicable). Investigate parser and fixture gaps with file/line evidence and attach them to issue #1; propose separately scoped source PRs only if justified.

## Engineering reference and follow-up
- Adopt only relevant guidance from https://arran4.github.io/blog/post/2026/055-suggested-coding-standards-for-repositories-and-agents/ and https://arran4.github.io/blog/post/2026/020-optional-dependency-injection-via-type-switched-variadic-args/ .
- Do not merge implementation automatically; request review and follow repository-specific instructions.
