package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAndExplicitRefresh(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "examples")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"man-tool.1", "convert.html"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var log bytes.Buffer
	if err := run([]string{"--root", root}, &log); err == nil || !strings.Contains(log.String(), "stale example:") {
		t.Fatalf("missing artifacts accepted: %v %s", err, log.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "man-tool.html")); !os.IsNotExist(err) {
		t.Fatal("check wrote missing output")
	}
	if err := run([]string{"--root", root, "--write"}, &log); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--root", root}, &log); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "man-tool.ast.json")
	if err := os.WriteFile(stale, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--root", root}, &log); err == nil {
		t.Fatal("stale artifact accepted")
	}
	data, _ := os.ReadFile(stale)
	if string(data) != "stale" {
		t.Fatal("check overwrote stale output")
	}
	blocked := filepath.Join(dir, "man-tool.validation.json")
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--root", root, "--write"}, &log); err == nil {
		t.Fatal("directory at late example output was accepted")
	}
	data, _ = os.ReadFile(stale)
	if string(data) != "stale" {
		t.Fatal("late output failure partially refreshed earlier examples")
	}
	if err := os.WriteFile(filepath.Join(dir, "man-tool.1"), []byte(".B \"unclosed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--root", root, "--write"}, &log); err == nil {
		t.Fatal("invalid source accepted")
	}
	data, _ = os.ReadFile(stale)
	if string(data) != "stale" {
		t.Fatal("failed build partially refreshed examples")
	}
}
