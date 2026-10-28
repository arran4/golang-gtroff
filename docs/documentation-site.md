# GitHub Pages: Actions artifact deployment

This site is built using Hugo Book, based on the site in [go-subcommand](https://github.com/arran4/go-subcommand).

## Documentation is maintained in one place

Edit `docs/*.md`. The `docs/scripts/stage-docs.py` script builds derived Hugo pages into `docs/content/docs/reference/` and adjusts links to other guides or GitHub source artifacts. The generated files are not committed.

The homepage and section indices are maintained separately under `docs/content/`. The Hugo Book theme is a pinned Git submodule.

## Test locally

```sh
git submodule update --init --recursive
python3 docs/scripts/stage-docs.py
hugo --source docs --minify --baseURL 'https://arran4.github.io/golang-gtroff/'
```

The generated `docs/public/` directory is ignored. Check navigation, images, code blocks and project-prefix links.

## Action-based deployment, not a branch deployment

The workflow is `.github/workflows/pages.yml`. It uses the **GitHub Actions** Pages artifact/deployment protocol:

1. On documentation PRs, build the Hugo site without deploying it.
2. On eligible main-branch documentation pushes (or manual dispatch), build once more.
3. Only when the repository is **public**, upload `docs/public/` with `actions/upload-pages-artifact`.
4. Use `actions/deploy-pages` with only `pages: write` and `id-token: write` in the deploy job.

**There is no `gh-pages` branch, no output pushed to any branch, and no binary GitHub Release publication.**

After reviewing the repository history and choosing to make it public, set **Settings → Pages → Build and deployment → Source: GitHub Actions**. An Action workflow alone does not configure the repository settings.

The target URL is https://arran4.github.io/golang-gtroff/ after the deployment actually succeeds; it is not live merely because the PR merged.

## Security and verification

A private repository must not accidentally publish a Pages site. The workflow gates both artifact upload and deployment on the repository being public. All PRs build without deployment, including external contributions.

Complete a full-history secret scan and licensing/attribution review before public visibility changes. If Actions quota prevents execution, report that the site has **not been verified**; don't equate a zero-step workflow failure with a Hugo error.
