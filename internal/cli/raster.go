package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/fontset"
	"github.com/arran4/golang-gtroff/man"
	"github.com/arran4/golang-gtroff/renderimage"
)

func renderRaster(e Environment, input, output string, node groff.NodeID, o renderimage.Options, paths struct{ regular, bold, italic, mono string }) int {
	// The CLI distinguishes an explicit invalid zero from the API's zero defaults.
	if o.Width < 128 || o.Margin < 0 || o.PageHeight < 128 || o.MaxHeight < 128 || o.FontSize < 6 || o.DPI < 36 || o.LineHeight < 1 || o.MaxPixels < 1 || o.Quality < 1 {
		return usage(e, "image dimensions, typography, quality and resource limits must be within documented ranges")
	}
	if err := renderimage.ValidateOptions(o); err != nil {
		return usage(e, err.Error())
	}
	if _, err := renderimage.OutputPaths(output, o.Format, 1); err != nil {
		return usage(e, err.Error())
	}
	if output != "-" && e.WriteOutput == nil {
		return failure(e, "output file writing is unavailable")
	}
	source := "<stdin>"
	sourcePath := ""
	reader := e.In
	var owned io.ReadCloser
	if input != "-" {
		source = input
		sourcePath = input
		if e.Open == nil {
			return failure(e, "file input is unavailable")
		}
		f, err := e.Open(input)
		if err != nil {
			return failure(e, "open "+strconv.Quote(input)+": "+err.Error())
		}
		reader = f
		owned = f
	}
	tree, err := groff.ParseAST(reader, groff.WithPackage("man"), groff.WithSourceName(source))
	if owned != nil {
		if closeErr := owned.Close(); closeErr != nil {
			return failure(e, "close input: "+closeErr.Error())
		}
	}
	if err != nil {
		return failure(e, "read input: "+err.Error())
	}
	doc, err := man.Build(tree, man.Options{})
	if err != nil {
		return failure(e, "semantic input: "+err.Error())
	}
	var selection *man.Selection
	switch {
	case o.Section != "":
		selection, err = doc.SelectSection(o.Section, false)
	case node != 0:
		n := tree.Node(node)
		if n == nil {
			err = fmt.Errorf("AST node %d not found", node)
		} else {
			if n.Kind() == groff.ASTInline {
				n = n.Parent()
			}
			selection, err = doc.SelectAST(n.ID())
		}
	default:
		selection, err = doc.SelectNode(1)
	}
	if err != nil {
		return usage(e, err.Error())
	}
	for _, face := range []struct {
		path string
		data *[]byte
	}{{paths.regular, &o.Fonts.Regular}, {paths.bold, &o.Fonts.Bold}, {paths.italic, &o.Fonts.Italic}, {paths.mono, &o.Fonts.Mono}} {
		if face.path == "" {
			continue
		}
		if e.Open == nil {
			return failure(e, "font file access is unavailable")
		}
		f, err := e.Open(face.path)
		if err != nil {
			return failure(e, "open font: "+err.Error())
		}
		data, readErr := io.ReadAll(io.LimitReader(f, fontset.MaxFaceBytes+1))
		closeErr := f.Close()
		if readErr != nil {
			return failure(e, "read font: "+readErr.Error())
		}
		if closeErr != nil {
			return failure(e, "close font: "+closeErr.Error())
		}
		if len(data) == 0 || len(data) > fontset.MaxFaceBytes {
			return failure(e, "font exceeds size limit or is empty")
		}
		*face.data = data
	}
	plan, err := renderimage.LayoutSelection(doc, selection, o)
	if plan != nil {
		if e := writeDiagnostics(e.Err, plan.Result().Diagnostics); e != nil {
			return ExitIO
		}
	}
	if err != nil {
		if errors.Is(err, renderimage.ErrStrict) {
			return ExitInvalid
		}
		return failure(e, "layout image: "+err.Error())
	}
	result := plan.Result()
	names, err := renderimage.OutputPaths(output, o.Format, result.Pages)
	if err != nil {
		if errors.Is(err, renderimage.ErrMultiplePages) {
			return usage(e, err.Error())
		}
		return failure(e, err.Error())
	}
	if len(names) > 1 && e.WriteOutputs == nil {
		return failure(e, "multi-page output writing is unavailable")
	}
	var files []OutputFile
	_, err = plan.Encode(func(page renderimage.Page, data []byte) error {
		files = append(files, OutputFile{Path: names[page.Number-1], Data: data})
		return nil
	})
	if err != nil {
		return failure(e, "encode image: "+err.Error())
	}
	switch {
	case output == "-":
		if err := writeBytes(e.Out, files[0].Data); err != nil {
			return failure(e, "write stdout: "+err.Error())
		}
	case len(files) == 1:
		if err := e.WriteOutput(files[0].Path, sourcePath, files[0].Data); err != nil {
			return failure(e, "write image: "+err.Error())
		}
	default:
		if err := e.WriteOutputs(sourcePath, files); err != nil {
			return failure(e, "write images: "+err.Error())
		}
	}
	_, code := assessment(result.Diagnostics)
	return code
}
