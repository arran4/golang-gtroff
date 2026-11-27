# Documentation website

The [gtroff documentation site](https://arran4.github.io/golang-gtroff/) uses Hugo Book. The original guides in `docs/*.md` are shared with the website; the staging script generates Hugo pages and rewrites links. Edit the original guides rather than the generated copies.

## Preview locally

Use Hugo extended 0.169.0, as pinned by the Pages workflow.

```sh
git submodule update --init --recursive
python3 docs/scripts/stage-docs.py
hugo --source docs --minify --baseURL 'https://arran4.github.io/golang-gtroff/'
```

Output is written to `docs/public/`. For local browsing use `hugo --source docs server`.

GitHub Pages builds documentation from `.github/workflows/pages.yml` and deploys eligible pushes to public `main`. The generated site is not committed to a separate branch.
