// Package releasecheck verifies candidate archives, binary metadata and checksums.
// It uses only the standard library and executes only the requested native binary.
package releasecheck

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type artifact struct{ Name, Path, Type, Goos, Goarch string }
type metadata struct {
	ProjectName           string `json:"project_name"`
	Version, Commit, Date string
}

const maxFile = 256 << 20

// Check validates the complete expected target set and every published checksum.
// Native smoke uses a binary extracted from the archive, never a rebuilt command.
func Check(dir, project string, native bool, out io.Writer) error {
	if project != "gtroff" && project != "gtroff-view" {
		return fmt.Errorf("unknown project %s", project)
	}
	var all []artifact
	var meta metadata
	if err := readJSON(filepath.Join(dir, "artifacts.json"), &all); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(dir, "metadata.json"), &meta); err != nil {
		return err
	}
	if meta.ProjectName != project || meta.Version == "" || len(meta.Commit) != 40 {
		return fmt.Errorf("invalid release metadata")
	}
	if _, err := time.Parse(time.RFC3339Nano, meta.Date); err != nil {
		return err
	}
	var checksum artifact
	archives := map[string]artifact{}
	packages := map[string]int{}
	published := map[string]artifact{}
	for _, a := range all {
		if a.Name == "" || path.Base(a.Name) != a.Name {
			return fmt.Errorf("unsafe artifact name %q", a.Name)
		}
		switch a.Type {
		case "Checksum":
			if checksum.Name != "" {
				return fmt.Errorf("duplicate checksum index")
			}
			checksum = a
		case "Archive":
			key := a.Goos + "/" + a.Goarch
			if _, ok := archives[key]; ok {
				return fmt.Errorf("duplicate target %s", key)
			}
			archives[key] = a
			published[a.Name] = a
		case "Linux Package":
			packages[a.Goarch+"/"+packageFormat(a.Name)]++
			published[a.Name] = a
		case "Source":
			published[a.Name] = a
		}
	}
	want := []string{"linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64", "darwin/amd64", "darwin/arm64"}
	if project == "gtroff-view" {
		want = []string{"linux/amd64"}
	}
	if len(archives) != len(want) {
		return fmt.Errorf("archive target count %d, need %d", len(archives), len(want))
	}
	if project == "gtroff" {
		for _, arch := range []string{"amd64", "arm64"} {
			for _, format := range []string{"deb", "rpm", "apk", "pkg.tar.zst"} {
				if packages[arch+"/"+format] != 1 {
					return fmt.Errorf("missing/duplicate package %s/%s", arch, format)
				}
			}
		}
	}
	if checksum.Name == "" {
		return fmt.Errorf("missing checksum index")
	}
	if err := checkHashes(dir, checksum.Name, published); err != nil {
		return err
	}
	for _, target := range want {
		a, ok := archives[target]
		if !ok {
			return fmt.Errorf("missing archive %s", target)
		}
		contents, err := archive(filepath.Join(dir, a.Name))
		if err != nil {
			return err
		}
		bin := project
		if a.Goos == "windows" {
			bin += ".exe"
		}
		data, ok := contents[bin]
		if !ok {
			return fmt.Errorf("%s lacks %s", a.Name, bin)
		}
		if err := inspect(data, a.Goos, a.Goarch); err != nil {
			return fmt.Errorf("%s: %w", target, err)
		}
		files := []string{"LICENSE", "fontset/LICENSE-Go-Fonts.txt", "renderimage/LICENSE-Go-Image.txt", "readme.md", "examples/man-tool.1"}
		if project == "gtroff" {
			files = append(files, "docs/generated/man/gtroff.1", "docs/generated/man/gtroff-render.1", "docs/generated/completions/gtroff.bash", "docs/generated/completions/_gtroff", "docs/generated/completions/gtroff.fish", "docs/generated/completions/gtroff.ps1", "examples/man-tool.html", "examples/man-tool.png", "examples/man-tool.ast.json")
		}
		for _, name := range files {
			if len(contents[name]) == 0 {
				return fmt.Errorf("%s lacks %s", a.Name, name)
			}
		}
		fmt.Fprintf(out, "verified archive %s (%s, CGO=0)\n", a.Name, target)
		if native && a.Goos == runtime.GOOS && a.Goarch == runtime.GOARCH {
			if err := smoke(data, contents["examples/man-tool.1"], project, meta); err != nil {
				return err
			}
			fmt.Fprintf(out, "native smoke passed: %s\n", target)
		}
	}
	if native {
		if _, ok := archives[runtime.GOOS+"/"+runtime.GOARCH]; !ok {
			return fmt.Errorf("no native candidate for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}
	return nil
}

func packageFormat(name string) string {
	for _, f := range []string{"pkg.tar.zst", "deb", "rpm", "apk"} {
		if strings.HasSuffix(name, "."+f) {
			return f
		}
	}
	return "unknown"
}
func readJSON(file string, v any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
func checkHashes(dir, name string, want map[string]artifact) error {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	scan := bufio.NewScanner(bytes.NewReader(data))
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		if len(fields) != 2 {
			return fmt.Errorf("invalid checksum line")
		}
		name := strings.TrimPrefix(fields[1], "*")
		if path.Base(name) != name || seen[name] {
			return fmt.Errorf("unsafe/duplicate checksum %q", name)
		}
		seen[name] = true
		if len(fields[0]) != 64 {
			return fmt.Errorf("invalid SHA-256")
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return err
		}
		file, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		hash := sha256.New()
		n, err := io.Copy(hash, io.LimitReader(file, maxFile+1))
		_ = file.Close()
		if err != nil {
			return err
		}
		if n > maxFile {
			return fmt.Errorf("artifact exceeds inspection budget")
		}
		if hex.EncodeToString(hash.Sum(nil)) != fields[0] {
			return fmt.Errorf("checksum mismatch %s", name)
		}
	}
	if err := scan.Err(); err != nil {
		return err
	}
	for name := range want {
		if !seen[name] {
			return fmt.Errorf("unchecked published artifact %s", name)
		}
	}
	return nil
}
func archive(file string) (map[string][]byte, error) {
	out := map[string][]byte{}
	total := int64(0)
	add := func(name string, r io.Reader) error {
		if !fs.ValidPath(name) || path.Clean(name) != name {
			return fmt.Errorf("unsafe archive member %q", name)
		}
		if _, exists := out[name]; exists {
			return fmt.Errorf("duplicate archive member %s", name)
		}
		data, err := io.ReadAll(io.LimitReader(r, maxFile-total+1))
		if err != nil {
			return err
		}
		total += int64(len(data))
		if total > maxFile {
			return fmt.Errorf("archive inspection budget exceeded")
		}
		out[name] = data
		return nil
	}
	if strings.HasSuffix(file, ".zip") {
		z, err := zip.OpenReader(file)
		if err != nil {
			return nil, err
		}
		defer z.Close()
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}
			if !f.Mode().IsRegular() {
				return nil, fmt.Errorf("nonregular archive member")
			}
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			err = add(f.Name, r)
			_ = r.Close()
			if err != nil {
				return nil, err
			}
		}
	} else {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			if h.Typeflag == tar.TypeDir {
				continue
			}
			if h.Typeflag != tar.TypeReg {
				return nil, fmt.Errorf("nonregular archive member")
			}
			if err := add(h.Name, tr); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
func inspect(data []byte, goos, arch string) error {
	info, err := buildinfo.Read(bytes.NewReader(data))
	if err != nil {
		return err
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["CGO_ENABLED"] != "0" || settings["GOOS"] != goos || settings["GOARCH"] != arch || settings["-trimpath"] != "true" {
		return fmt.Errorf("unexpected Go build settings: %v", settings)
	}
	switch goos {
	case "linux":
		f, err := elf.NewFile(bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer f.Close()
		want := elf.EM_X86_64
		if arch == "arm64" {
			want = elf.EM_AARCH64
		}
		if f.Machine != want {
			return fmt.Errorf("ELF architecture does not match %s", arch)
		}
		for _, p := range f.Progs {
			if p.Type == elf.PT_INTERP {
				return fmt.Errorf("Linux binary needs a dynamic interpreter")
			}
		}
		if libs, err := f.ImportedLibraries(); err != nil || len(libs) > 0 {
			return fmt.Errorf("Linux shared libraries: %v (%v)", libs, err)
		}
	case "windows":
		f, err := pe.NewFile(bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer f.Close()
		want := uint16(pe.IMAGE_FILE_MACHINE_AMD64)
		if arch == "arm64" {
			want = pe.IMAGE_FILE_MACHINE_ARM64
		}
		if f.Machine != want {
			return fmt.Errorf("PE architecture does not match %s", arch)
		}
	case "darwin":
		f, err := macho.NewFile(bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer f.Close()
		want := macho.CpuAmd64
		if arch == "arm64" {
			want = macho.CpuArm64
		}
		if f.Cpu != want {
			return fmt.Errorf("Mach-O architecture does not match %s", arch)
		}
	default:
		return fmt.Errorf("unsupported executable OS %s", goos)
	}
	return nil
}
func smoke(binary, source []byte, project string, m metadata) error {
	dir, err := os.MkdirTemp("", "gtroff-release-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	name := project
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	exe := filepath.Join(dir, name)
	if err := os.WriteFile(exe, binary, 0700); err != nil {
		return err
	}
	run := func(input string, args ...string) ([]byte, int, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, exe, args...)
		c.Dir = dir
		c.Stdin = strings.NewReader(input)
		c.Env = append(os.Environ(), "PATH=")
		var out, stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr
		err := c.Run()
		if err == nil && stderr.Len() != 0 {
			return nil, 0, fmt.Errorf("%v: unexpected diagnostics %s", args, stderr.String())
		}
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				return nil, 0, err
			}
		}
		return out.Bytes(), code, nil
	}
	versionArg := "version"
	if project == "gtroff-view" {
		versionArg = "--version"
	}
	version, code, err := run("", versionArg)
	if err != nil || code != 0 {
		return fmt.Errorf("version smoke: %d %v", code, err)
	}
	date, _ := time.Parse(time.RFC3339Nano, m.Date)
	want := fmt.Sprintf("%s %s (commit %s, built %s)\n", project, m.Version, m.Commit, date.UTC().Format(time.RFC3339))
	if string(version) != want {
		return fmt.Errorf("version mismatch: got %q want %q", version, want)
	}
	help, code, err := run("", "--help")
	if err != nil || code != 0 || !bytes.Contains(help, []byte("Usage:")) {
		return fmt.Errorf("help smoke failed: %d %v", code, err)
	}
	if project == "gtroff-view" {
		return nil
	}
	for _, args := range [][]string{{"validate", "--strict"}, {"render", "--format", "html"}, {"render", "--format", "png", "--single"}} {
		data, code, err := run(string(source), args...)
		if err != nil || code != 0 || len(data) == 0 {
			return fmt.Errorf("%v smoke: %d %v", args, code, err)
		}
		if args[0] == "render" && args[2] == "html" && !bytes.Contains(data, []byte("data:font/ttf;base64,")) {
			return fmt.Errorf("HTML lacks embedded fonts")
		}
		if args[0] == "render" && args[2] == "png" {
			if _, err := png.Decode(bytes.NewReader(data)); err != nil {
				return err
			}
		}
	}
	invalid, code, err := run(".UNKNOWN retained\n", "validate", "--strict", "--format", "json")
	if err != nil || code != 1 || !bytes.Contains(invalid, []byte(`"status":"invalid"`)) {
		return fmt.Errorf("negative artifact smoke did not reject unknown syntax")
	}
	return nil
}
