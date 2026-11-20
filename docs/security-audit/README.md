# Public-history and artifact audit checkpoint

Evidence for [#75](https://github.com/arran4/golang-gtroff/issues/75),
2026-11-13. **No real credential exposure was confirmed in the examined
inputs.** All observed scanner candidates were reviewed false positives.
This is a dated, bounded audit checkpoint, not release/publication clearance.
The earlier policy described intended auditing without recording its results;
`SECURITY.md` now distinguishes policy from verified evidence.

## Scope and findings

The initial accessible snapshot includes public `main`
`83e89c33b73b7cc8153f5d05733c61cd7774baad`, #77
`4cbe44b5ed5cc713ce73a827b50250bfa68e790c`, #64
`a5359412d0363ade8b7dd611e365a79412397b98`, and the live feature branches.
[All 91 fetched refs](refs.txt) identify the exact snapshot: 22 branch refs,
52 PR heads and 17 PR merge refs, with no tags. The fresh mirror is
non-shallow, has 189 reachable commits and passes `git fsck --full`.
All 52 PR heads returned by the PR API are present. No accessible forks,
release records or discussion attachment URLs were returned.

| Input | Examined bytes/records | Gitleaks candidates | TruffleHog candidates |
| --- | --- | ---: | ---: |
| Full accessible Git history, including merge diffs | 189 commits / 91 refs | 6 | 0 |
| Every reachable Git blob, independently exported | 1,715 blobs / 18,007,228 bytes | 6 | 0 |
| Issue/PR bodies and discussions | 100 bodies, 104 issue comments; 0 inline comments or review submissions | 0 | 0 |
| Every available workflow log bundle in the initial API inventory | 100 ZIPs, 248 extracted files / 2,394,425 bytes | 0 | 0 |
| All three retained candidate artifacts and expanded payloads | 75 outer members plus 3,494 inner regular files / 641,562,957 expanded inner bytes | 21 | 0 |

The counts overlap and are **not counts of distinct incidents**. All 33
Gitleaks occurrences refer to the same ordinary structural-documentation
sentence in `docs/roadmap-verification.md:35`, appearing in six historical
blobs and seven source/platform archives per retained candidate. Each exact
line was inspected. It describes typed attributes, tokens and tree
relationships, rather than assigning a credential. The
`generic-api-key` detector interpreted the prose as a key pattern. No
allowlist or suppression was added; the observations and classifications
remain in [scans.json](scans.json).

TruffleHog ran with online credential verification and update checks disabled.
No candidate credentials were sent to a provider for validation. Zero
candidates means none were reported by these configured detectors, not that
every conceivable secret was disproved. The all-blob scan supplements the
diff scans, including binary blobs and merge-only content. No scan errors or
unexpected warning diagnostics were observed in these runs.

The [summary](summary.json), [scanner identities](scanners.json),
[artifact identities](artifacts.json), [log bundle identities](logs.json)
and [metadata-only receipts](scans.json)
contain no raw matches or diagnostic bodies. Candidate ordering can vary with
scanner concurrency; compare coverage and candidate metadata rather than
requiring byte-identical scanner execution order. Final PR-head/ref refreshes
are recorded separately in the PR/issue verification comments, so the evidence
commit is not falsely described as having scanned its own future commit SHA.

## Actual retained artifacts and manual checks

| Artifact ID / workflow | Workflow head | Built PR merge commit |
| --- | --- | --- |
| 14011196215 / [43366634988](https://github.com/arran4/golang-gtroff/actions/runs/43366634988) | `be4e2f8ed1baba939062659c51971d762164646d` (#96) | `6126edfcae84d6acac47cf4a429424f2929bce53` |
| 14011458912 / [43368765913](https://github.com/arran4/golang-gtroff/actions/runs/43368765913) | `b4c1d23e93870958cb3178ebb333365bb88986cc` (#98) | `5a41c3c95be9048dd6c467a7279c7711cec6cd9c` |
| 14013535198 / [43371865182](https://github.com/arran4/golang-gtroff/actions/runs/43371865182) | `0d4064b04f760a556cb34929249aa8e6ec85d5a1` (#100) | `fe21e3b9e1dc40ee132e6aa5be052d90804c23b9` |

All built commits are present in the fetched refs. These are the downloaded
original Actions ZIPs, not rebuilt substitutes. Their SHA-256 hashes and sizes
are in `artifacts.json`. Each contains 25 outer members: source and six
platform archives, eight Linux packages, six direct binaries and metadata.

The inspection expanded 57 nested containers: per candidate, one source
archive, six platform archives, eight Debian/RPM/APK/Arch packages and four
Debian control/data archives. The ZIP, tar/gzip, ar, RPM/cpio, APK and Zstd
payloads were inspected as data. No package script was installed or executed.
Extraction used bsdtar/libarchive 3.7.4 with explicit member/path/type and
resource checks; no nonregular members were skipped in these inputs.

Sixty binary copies represent 18 distinct binaries across the three
candidates. Their Go build metadata reports Go 1.26.10, `CGO_ENABLED=0`,
`-trimpath=true` and `vcs.modified=false`. Targeted checks found no build-host
path prefixes `/home/user/`, `/workspace/`, `/home/runner/work/` or `/Users/`
in those bytes. This is a targeted privacy check, not proof that all possible
internal details are absent. No binary was executed for this inspection.

A supplementary marker review of UTF-8 Git blobs without NUL bytes found no
examined private-host/private-key markers or RFC1918 addresses. Two historical
blobs contain a synthetic user-info URL in
`renderhtml/render_test.go:153`, `TestURLPolicy`; it uses a local/reserved
host and is explicitly in the rejected-URL test set. The synthetic fixture
was retained. Published maintainer contact information in the security policy
is intentional project metadata, not a credential finding. The committed
viewer smoke image was visually checked against its synthetic MAN-TOOL page;
this inspection does not claim OCR coverage of every historical image.

Known GNU reference prose, Go fonts, adapted Go Image code and the environ
fixture remain subject to their original terms and the separate
[#67/#81 assessment](../provenance/README.md). Secret detection cannot establish
copyright ownership or absence of copied proprietary expression.

## Settings observed without changes

- Repository visibility: public; default branch: `main`; `has_pages=false`.
  No live Pages publication was established by this audit.
- Private vulnerability reporting: **disabled**, confirmed by its read API.
- Dependabot vulnerability alerts: **enabled**, confirmed through GraphQL;
  the alert API returned no records at this checkpoint.
- Default branch's GraphQL `branchProtectionRule`: null; repository rulesets:
  empty. No tags were returned. These observations do not establish a
  deliberate maintainer decision about branch/release-tag protection.
- Secret-scanning alert and branch-protection REST reads returned **403**
  (“Resource not accessible by personal access token”). Repository metadata
  did not return `security_and_analysis`. Secret scanning and push protection
  therefore remain **unverified**, rather than assumed enabled or disabled.

No visibility, protection, vulnerability-reporting, scanner or Dependabot
setting was changed. Configuration requiring authority is a maintainer review
item; an empty alert list alone is not a configuration check.

## Reproduce a new checkpoint

Use an explicit, private `AUDIT_DIR` outside the checkout, with mode 0700 and
`umask 077`. Never upload raw exports, ZIP contents or scanner output. The
following is verification tooling, with no new requirement for ordinary Go
library/CLI use.

1. Obtain the pinned Linux amd64 releases linked in `scanners.json`, compare
   their archives against the release checksum manifests, extract only the
   named scanner executables, and record archive/binary hashes. This audit
   checked those manifests; detached signatures were **not** verified.
2. Create and enumerate a fresh full mirror:

   ```sh
   git clone --mirror https://github.com/arran4/golang-gtroff.git "$AUDIT_DIR/repository.git"
   git -C "$AUDIT_DIR/repository.git" fetch origin \
     '+refs/pull/*/head:refs/pull/*/head' \
     '+refs/pull/*/merge:refs/pull/*/merge'
   git -C "$AUDIT_DIR/repository.git" rev-parse --is-shallow-repository
   git -C "$AUDIT_DIR/repository.git" for-each-ref \
     --format='%(objectname) %(refname)' > "$AUDIT_DIR/refs.txt"
   git -C "$AUDIT_DIR/repository.git" rev-list --all --count
   git -C "$AUDIT_DIR/repository.git" fsck --full --no-reflogs
   ```

3. Export all reachable blob objects using `git rev-list --objects --all
   --no-object-names`, `git cat-file --batch-check` and `git cat-file blob`.
   Use each hexadecimal object ID as the filename, and retain an inventory
   of object IDs/sizes. The observed export bounded individual blobs to
   64 MiB and total bytes to 512 MiB; exceeding a limit must stop the audit.
4. With read-only `gh api --paginate` requests, export all issue/PR bodies,
   issue comments, inline review comments, each PR's review submissions,
   releases and accessible forks. Keep only the needed bodies/titles/IDs/URLs
   in restricted local inputs. Endpoint families are `issues?state=all`,
   `issues/comments`, `pulls/comments`, `pulls?state=all`,
   `pulls/NUMBER/reviews`, `releases` and `forks`. Record every page/count and
   inspect any attachment URLs rather than silently omitting them.
5. Enumerate `actions/runs` and `actions/artifacts` with pagination. Download
   every accessible run's `/logs` ZIP and every retained artifact's `/zip`;
   record IDs, workflow heads, availability and SHA-256 hashes. Store API
   error bodies privately rather than printing them. Record unavailable
   originals as a coverage limit, not as clean files.
6. Expand ZIP log/artifact files and the nested formats listed above in owned
   private directories. Validate names and member types before writing;
   reject traversal, symlink and conflicting-path entries. For RPM's absolute
   member names, map them beneath the private root; never install to `/usr`.
   The observed extraction read regular member payloads with
   `bsdtar -xOf ARCHIVE -- MEMBER`, then wrote validated local paths. It
   bounded a member to 128 MiB, nesting to four inner levels, an archive to
   20,000 entries and expanded inner bytes to 2 GiB. Outer ZIP/log limits
   were separately checked. Any skipped/unsupported member or exceeded
   limit needs explicit incomplete-coverage reporting. Preserve package
   metadata and scripts as data, without executing them.
7. Build and run the metadata-only Go wrapper on each prepared scope:

   ```sh
   go build -o "$AUDIT_DIR/bin/publicaudit" ./internal/publicaudit
   "$AUDIT_DIR/bin/publicaudit" -scanner gitleaks \
     -executable "$AUDIT_DIR/bin/gitleaks" -kind git \
     -source "$AUDIT_DIR/repository.git" -label history \
     > "$AUDIT_DIR/history-gitleaks.json"
   "$AUDIT_DIR/bin/publicaudit" -scanner trufflehog \
     -executable "$AUDIT_DIR/bin/trufflehog" -kind git \
     -source "$AUDIT_DIR/repository.git" -label history \
     > "$AUDIT_DIR/history-trufflehog.json"
   ```

   Repeat both scanners with `-kind filesystem` for the reachable-blob,
   discussion, expanded-log and expanded-artifact directories, using a
   distinct label/receipt for each. Review candidates and metadata locally
   before publishing only the redacted evidence.
8. Read security settings independently; record positive configuration
   evidence and denied/unsupported reads separately. Repeat on the final
   planned integrated release candidate and latest public refs, not only an
   earlier feature checkpoint.

The exact historical artifacts can only be re-inspected while their original
bytes remain available. Re-running after refs, discussions, tools or retained
artifacts change produces a **new** checkpoint, not identical historical
coverage. Rebuilt equivalents do not prove that expired originals were clean.

### Wrapper guarantees and limits

[`internal/publicaudit`](../../internal/publicaudit/main.go) requires explicit
local inputs/executables and verifies the pinned version strings. It uses
Gitleaks' built-in rules via a private temporary config/empty ignore file,
ignores inline suppression comments, and requests all Git refs plus separate
merge diffs. TruffleHog uses the controlled bare mirror directly with no
head/base filter, avoiding an extra clone that could lose PR refs. Do not use
that trusted-local-config mode on an unreviewed existing repository config.
The caller must still verify that the supplied mirror is fresh and non-shallow.

The wrapper disables TruffleHog network verification/update checks, clears
scanner/config environment overrides, and emits only rule, relative file,
line and commit metadata. Match/Secret/Raw/Redacted/ExtraData, author/message
fields and arbitrary diagnostic bodies are discarded in process memory.
Candidate paths/rule names still require review before public posting.

It rejects malformed output, inconsistent finding exit codes and unexpected
diagnostics as incomplete. Capture limits are 64 MiB stdout, 16 MiB stderr,
16 MiB per TruffleHog JSONL record, with a 15-minute execution context. An
overflow cancels the child. Archive/decoder limits remain finite and must be
considered alongside the explicit extraction inventory. A zero wrapper exit
means the requested tool executions finished, **not** that candidate lists
are empty or that a release is approved. It is not an automatic CI clearance
gate, arbitrary-output sanitizer or hard operating-system sandbox.

Unit tests exercise candidate-field removal, diagnostic failures, pin
mismatch, all-ref/no-verification arguments, path metadata and output-cap
enforcement through `io.Copy`. Full Go/race/vet and generation checks passed;
actual pinned executables ran on all five prepared scopes. Windows amd64 and
macOS arm64 wrapper tests compile; those scanners were executed on Linux,
not those other platforms.

## Remaining release gates

Keep #75 open until the final integrated release candidate/latest public
refs are checked, inaccessible security settings are verified by an authorized
maintainer, and any deliberate protection/reporting decisions are recorded.
Only three candidate artifacts were retained under the workflow's one-day
retention; this audit cannot establish the original bytes of older candidates.
There is no final integrated candidate for the still-open workbench PR forest.

The audit covers accessible repository history/PR refs, not an exhaustive
enumeration of unknown unreferenced GitHub objects, private/deleted forks or
the full histories of external dependencies/submodules. Selected manual
privacy/content checks supplement pattern-based scanning; they do not prove
universal absence of sensitive/proprietary material. #67/#81 rights decisions
and #29 release engineering remain independent. No merge, release, tag,
relicensing or security-setting change was performed.
