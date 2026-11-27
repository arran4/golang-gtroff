# Building and installing gtroff

The headless command builds for Linux, Windows and macOS on amd64 and arm64. It uses bundled fonts and needs neither system groff nor CGO. The optional Linux desktop viewer has its own build and display requirements.

Build from a checkout with Go 1.26 or newer:

```sh
go build -o gtroff ./cmd/gtroff
go install ./cmd/gtroff
```

## Release archives and packages

Check [GitHub Releases](https://github.com/arran4/golang-gtroff/releases) for available downloads. The packaging configuration produces headless archives and Debian, RPM, APK and Arch Linux packages. The Linux amd64/X11 viewer uses a separate archive.

Verify the downloaded archive against its release checksum before extracting it. For example (replace VERSION with the release version):

```sh
sha256sum --ignore-missing -c gtroff_VERSION_checksums.txt
tar -xzf gtroff_VERSION_linux_amd64.tar.gz
sudo install -m 0755 gtroff /usr/local/bin/gtroff
gtroff version
```

On macOS use `shasum -a 256`; on Windows use PowerShell's `Get-FileHash -Algorithm SHA256` and extract the ZIP archive. macOS and Windows builds are not advertised as signed distributions.

Linux packages install the binary, manual pages, completion scripts, notices and example source.

```sh
sudo dpkg -i gtroff_VERSION_linux_amd64.deb
man gtroff-render
```

## Man pages and shell completions

```sh
gtroff man > gtroff.1
gtroff man render > gtroff-render.1
gtroff completion bash > gtroff.bash
gtroff completion zsh > _gtroff
gtroff completion fish > gtroff.fish
gtroff completion powershell > gtroff.ps1
```

Bash completions can be sourced directly. Zsh uses its usual `fpath`/compinit mechanism; Fish reads its completion directory and PowerShell can dot-source its script. Completions suggest commands, flags and filenames, not selector or flag-value enums.

## Developer candidate builds

The CI configuration is in `.github/workflows/ci.yaml`. GoReleaser builds candidates; the workflow publishes the verified assets for a version tag. Snapshot builds do not publish a release.

Use the GoReleaser version pinned in CI:

```sh
go mod tidy
go -C desktop mod tidy
go generate ./...
go test ./...
goreleaser check
goreleaser release --snapshot --clean
go run ./cmd/releasecheck --dist dist --native
```

The release checker inspects archive contents, architecture, checksums, build metadata and native executable behaviour. The optional viewer uses `.goreleaser-view.yaml` and `dist-view/`:

```sh
goreleaser release --config .goreleaser-view.yaml --snapshot --clean
go run ./cmd/releasecheck --dist dist-view --viewer --native
```

On Linux/X11, test the extracted viewer binary from the desktop module using Xvfb, xauth and xdotool:

```sh
cd desktop
CGO_ENABLED=0 xvfb-run -a -s '-screen 0 1600x1200x24 -nolisten tcp' go test ./cmd/gtroff-view -run '^TestX11Executable$' -count=1 -gui-smoke -gui-binary /absolute/path/to/gtroff-view
```

See [viewer requirements](viewer.md) and [licensing and attribution](licensing.md).
