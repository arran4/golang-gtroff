# Query and extraction contract v1

This recipe version is 1.0.0. It expects command-inventory schemaVersion 1,
query/extract JSON schemaVersion 1, and the capabilities `stateless-query`,
`structural-selectors`, `record-extraction`, `source-provenance`, `bounded-includes`.
Inspect the actual binary; tool version `dev` is not a compatibility guarantee.
The exported manifest records file hashes and these requirements without assuming
an agent vendor, configuration directory or proprietary invocation protocol.

## Select useful structure

```sh
gtroff query --format json 'section[name="ENVIRONMENT"] definition' document.1
gtroff query --format json 'definition[term="TOOL_HOME"]' document.1
gtroff query --view syntax --format json 'macro[macro=".B"][arg.0="TOOL_HOME"]' document.1
gtroff extract --format json 'definition[term="TOOL_HOME"]' document.1
gtroff extract --format csv --fields term,description,file,line definition document.1
gtroff extract --format raw 'section[name="ENVIRONMENT"]' document.1
```

Commands are argv examples: pass a whole selector as one argument; shell quotes
are syntax for an interactive shell, not part of the selector value. Flags precede
positional input in the documented CLI contract. Use `--` before a dash filename.
A document argument of `-` explicitly reads stdin; there is at most one document root per command.

Kinds include source/document/section/subsection/definition/term/paragraph and
syntax line/inline/macro/request/comment/unknown/block. `item` aliases definition.
The default semantic view groups common man content; choose `--view syntax` to
inspect physical syntax independently of interpretation. A source wrapper is one
file occurrence, not an expanded serialization.

`A B` means descendant and `A > B` means child. Match exact case-sensitive values
with `[field="value"]`, inequality with `!=`, contains with `*=`, prefix with `^=`
and suffix with `$=`. Numeric location fields also support `<`, `<=`, `>` and `>=`.
Use `normalized-text` for explicit case/whitespace-normalized matching. Relevant
fields include `name`, `term`, `text`, `raw`, `macro`, `arg.0`, `source`, `source-id`,
`include-depth`, `line`, `column`, `offset` and `end-offset`. This conservative
language has no SQL, loops, regex, pseudo-classes or implicit procedural state.

Selectors such as `source[source-id=1] > document > section[name="ENVIRONMENT"]`
can narrow ownership. Included fragments appear beneath their referring owner,
while semantic/font/heading interpretation remains local to each physical file.
No combined groff runtime state is simulated across includes. Validate a small
selection before expanding it to a collection; bounds and cancellation may return
a useful prefix rather than a complete inventory.

## Choose and validate output

| Mode | Meaning |
| --- | --- |
| query JSON | Full query report with diagnostics, Complete, Error and optional InputError |
| query JSONL | Self-contained match rows; not a global completion/diagnostic envelope |
| extract JSON | Full record report with Complete, QueryComplete, errors and diagnostics |
| extract JSONL | Summary envelope plus record envelopes, each with schemaVersion 1 |
| raw | Exact selected original byte spans; overlaps can repeat bytes |
| text | Interpreted semantic or syntax text, depending on view |
| grep | File/line/column records |
| CSV/TSV | Requested fields; choose term,description,file,line for definitions |

Match/record indices are one-based and ordered within one execution. Query IDs,
source IDs and semantic/node IDs are snapshot-local. Requery after a source edit;
these ordinals are not persistent cursors. JSON/JSONL are distinct protocols:
parse JSONL records individually, not as one JSON document. Text fields may retain
newlines; parse quoted CSV/TSV with a proper delimited-record reader rather than
splitting on line boundaries. An exit status and a
summary envelope carry facts absent from bare selected rows.

`Complete` describes enumeration/construction over the supplied snapshot, not
successful input EOF, valid syntax, resolved includes or rendering support.
Check each source's `complete`, source diagnostics, InputError/QueryError/Error
and report completion separately. Empty successful selection can mean no matches;
an incomplete/error result cannot prove absence. Unsupported/opaque blocks and
unknown requests retain source/provenance and warnings rather than being dropped.

Query normalization folds case/whitespace only when explicitly requested by a
predicate/output option. Extraction normalized text folds whitespace while
preserving case. Semantic text, exact raw roff and intentionally canonical output
are different representations. JSON string encoding replaces invalid UTF-8;
use raw bytes where byte fidelity matters. Extraction terms/descriptions and
styled blocks are source-backed data, not proof of full GNU groff semantics.

## Cite and preserve source ownership

Cite a filename plus original line/column or `[start,end)` byte interval; include
the source SHA-256 when the specific snapshot matters. Byte offsets are not rune
indices. Do not cite a flattened include expansion as if it came from the root.
Repeated include occurrences have different IDs but may refer to the same
physical name/span. Preserve occurrence evidence unless the task explicitly calls
for physical deduplication. Heading context and unsupported flags help explain a
fragment, while raw bytes remain the primary exact-source evidence.

The high-level source loader confines named inputs and local literal .so/.mso
references to an allowed root. Missing/denied/cyclic/disabled references stay
visible. No system macro search, network fetching or macro execution occurs.
Low-level syntax-only parsing does not implicitly open include files. For stdin,
choose an explicit allowed root only when external source loading is authorized.

Exit codes for query/extract: 0 permissive report, 1 strict diagnostics,
2 invalid CLI/selector/configuration, 3 input/output/resource/cancellation failure.
A nonzero result may retain a useful prefix; explain its limits. For file output,
`extract -o FILE` is explicit, protects loaded-source aliases and withholds output
on input/strict/serialization failure. The recipe itself does not authorize a
new destination. Other commands' policies must come from their current help.
