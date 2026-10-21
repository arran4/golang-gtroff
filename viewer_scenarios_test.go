package groff_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"image"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/arran4/golang-gtroff/viewer"
	"golang.org/x/tools/txtar"
)

//go:embed testdata/viewer/*.txtar
var viewerCases embed.FS
var updateViewer = flag.Bool("update-viewer", false, "explicitly update deterministic viewer controller scenarios")

func TestViewerScenarios(t *testing.T) {
	paths, e := fs.Glob(viewerCases, "testdata/viewer/*.txtar")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "testdata/viewer/"), func(t *testing.T) {
			raw, e := viewerCases.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			ar := txtar.Parse(raw)
			files := map[string][]byte{}
			for _, f := range ar.Files {
				if _, ok := files[f.Name]; ok {
					t.Fatal("duplicate name")
				}
				files[f.Name] = f.Data
			}
			var tc struct {
				Pages    []image.Point
				Viewport image.Point
				Actions  []viewer.Action
			}
			if e := json.Unmarshal(files["scenario.json"], &tc); e != nil {
				t.Fatal(e)
			}
			c, e := viewer.NewController(tc.Pages, tc.Viewport)
			if e != nil {
				t.Fatal(e)
			}
			type step struct {
				State                viewer.State
				Content, Destination image.Rectangle
				Changed              bool
				Error                string `json:",omitempty"`
			}
			var steps []step
			for _, a := range tc.Actions {
				before := c.State()
				var changed bool
				var err error
				if a.Kind == "click" {
					changed, err = c.Click(a.X, a.Y)
				} else {
					changed, err = c.Apply(a)
				}
				s := step{State: c.State(), Content: c.Content(), Destination: c.Destination(), Changed: changed}
				if err != nil {
					s.Error = err.Error()
					if c.State() != before {
						t.Fatal("invalid gesture changed state")
					}
				}
				if s.State.Zoom < viewer.MinZoom || s.State.Zoom > viewer.MaxZoom || s.State.PanX < 0 || s.State.PanY < 0 || s.State.Page < 0 || s.State.Page >= s.State.Pages {
					t.Fatal("state bounds escaped")
				}
				steps = append(steps, s)
			}
			got, e := json.MarshalIndent(steps, "", "  ")
			if e != nil {
				t.Fatal(e)
			}
			got = append(got, '\n')
			if *updateViewer {
				next := &txtar.Archive{Comment: ar.Comment}
				for _, f := range ar.Files {
					if f.Name != "expected.json" {
						next.Files = append(next.Files, f)
					}
				}
				next.Files = append(next.Files, txtar.File{Name: "expected.json", Data: got})
				if e := os.WriteFile(path, txtar.Format(next), 0644); e != nil {
					t.Fatal(e)
				}
			} else if !bytes.Equal(got, files["expected.json"]) {
				t.Errorf("controller expectation differs; approve refresh with -update-viewer\n%s", got)
			}
		})
	}
}
