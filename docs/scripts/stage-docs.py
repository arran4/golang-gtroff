#!/usr/bin/env python3
"""Stage the maintained docs/*.md guides as Hugo Book content.

Do not commit the generated content; docs/*.md remains the only source of truth.
"""
from __future__ import annotations

import json
import re
from pathlib import Path
from urllib.parse import urlsplit

DOCS = Path(__file__).resolve().parents[1]
ROOT = DOCS.parent
DEST = DOCS / "content" / "docs" / "reference"
REPO = "https://github.com/arran4/golang-gtroff/blob/main/"

# Link destinations in the original docs refer to other Markdown files or to
# repository sources. On the site, link local guides to their Hugo paths.
LINK = re.compile(r"(\!?\[[^\]]*\]\()([^)]+)(\))")


def link_target(target: str, source: Path, names: set[str]) -> str:
    path, sep, fragment = target.partition("#")
    if not path or path.startswith(("http:", "https:", "mailto:", "#", "/")):
        return target
    if path.endswith(".md") and Path(path).name in names and "/" not in path:
        return "../" + Path(path).stem + "/" + (sep + fragment if sep else "")
    resolved = (source.parent / path).resolve()
    if resolved.is_relative_to(ROOT.resolve()) and resolved.is_file():
        rel = resolved.relative_to(ROOT.resolve()).as_posix()
        return REPO + rel + (sep + fragment if sep else "")
    return target


def main() -> None:
    files = sorted(DOCS.glob("*.md"))
    DEST.mkdir(parents=True, exist_ok=True)
    names = {p.name for p in files}
    # Remove only generated sibling *.md; the maintained _index.md is untouched.
    for old in DEST.glob("*.md"):
        if old.name != "_index.md":
            old.unlink()
    for i, source in enumerate(files, 1):
        original = source.read_text(encoding="utf-8")
        heading = next((m.group(1).strip() for line in original.splitlines()
                        if (m := re.match(r"^# (.+)$", line))), source.stem)
        rewritten = LINK.sub(
            lambda match: match.group(1)
            + link_target(match.group(2), source, names)
            + match.group(3),
            original,
        )
        header = (
            "---\n"
            + "title: " + json.dumps(heading) + "\n"
            + f"weight: {i}\n"
            + "---\n\n"
            + "<!-- Generated from docs/" + source.name + "; edit the source file. -->\n\n"
        )
        (DEST / source.name).write_text(header + rewritten, encoding="utf-8")
    print(f"Staged {len(files)} maintained guides for Hugo")


if __name__ == "__main__":
    main()
