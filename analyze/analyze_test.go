package analyze_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/arran4/golang-gtroff/analyze"
	"github.com/arran4/golang-gtroff/source"
)

func readSet(t *testing.T, input string) *source.Set {
	t.Helper()
	set, err := source.Read(context.Background(), strings.NewReader(input), "input.1", source.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return set
}
func TestAnalysisStagesAndRequirements(t *testing.T) {
	input := ".TH TOOL 1\n.SH DESCRIPTION\nText \\fBbold\\fP \\*[STRING] \\n[REG] \\(zz.\n.nr REG 1\n.ds STRING value\n.if 1 .B conditional\n.sy never-run\n.TS\nl.\ndata\n.TE\n.MR printf 3\n.UNKNOWN literal\n"
	set := readSet(t, input)
	report, err := analyze.Run(context.Background(), set, analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || report.Profile.Name != "man" || report.Profile.Confidence != "high" {
		t.Fatalf("profile missing: %+v", report.Profile)
	}
	flags := report.Flags
	if !flags.HasConditional || !flags.RequiresEvaluation || !flags.UsesPreprocessor || !flags.HasDefinitions || !flags.HasReferences || !flags.HasSideEffectSyntax || !flags.HasUnknown || !flags.HasOpaque {
		t.Fatalf("feature flags missing: %+v", flags)
	}
	if report.Counts.Controls[".nr"] != 1 || report.Counts.Escapes["StringRef"] != 1 || !report.HasRule("builtin.preprocessor") {
		t.Fatal("deterministic aggregate inventory missing")
	}
	var title, unknown, conditional, table, boolFont bool
	for _, finding := range report.Findings {
		node := set.File(finding.Source).AST().Node(finding.Node)
		if node == nil || node.Span() != finding.Span || finding.Span.Start.Filename != "input.1" || finding.Reason == "" {
			t.Fatal("finding lost source evidence")
		}
		if finding.Capability.Rendering != analyze.NotAssessed {
			t.Fatal("syntax recognition became a rendering claim")
		}
		if finding.Rule == "builtin.control" {
			switch finding.Name {
			case ".TH":
				title = true
				if finding.Capability.Syntax != analyze.Parsed || finding.Capability.Semantics != analyze.Modeled {
					t.Fatal("title classification", finding)
				}
			case ".UNKNOWN":
				unknown = true
				if finding.Capability.Syntax != analyze.Unknown || finding.Capability.Semantics != analyze.Unsupported {
					t.Fatal("unknown classification", finding)
				}
			}
		}
		if finding.Feature == "conditional" {
			conditional = true
			if finding.Capability.Syntax != analyze.Parsed || finding.Capability.Semantics != analyze.Unsupported {
				t.Fatal("conditional was falsely evaluated", finding)
			}
		}
		if finding.Feature == "preprocessor" {
			table = true
			if finding.Name != "tbl" || finding.Capability.Semantics != analyze.Unsupported {
				t.Fatal("captured table became modeled layout", finding)
			}
		}
		if finding.Feature == "escape" && finding.Target == "B" {
			boolFont = true
			if finding.Capability.Semantics != analyze.Modeled {
				t.Fatal("known font not modeled")
			}
		}
	}
	if !title || !unknown || !conditional || !table || !boolFont {
		t.Fatal("expected evidence absent")
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	again, err := analyze.Run(context.Background(), set, analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := json.Marshal(again)
	if err != nil || string(encoded) != string(repeat) {
		t.Fatal("analysis is stateful/nondeterministic")
	}
	if set.File(1).AST().Source() != input {
		t.Fatal("analysis changed source")
	}
}
func TestStoredSyntaxAndCommentsAreNotExecution(t *testing.T) {
	input := ".ig\n.TH STORED 1\n.sy never-run\n.if 1 dangerous\n.so secret\n..\n.\\\" .TH COMMENT 1\ntext \\\" \\*[AFTER_COMMENT]\n"
	report, err := analyze.Run(context.Background(), readSet(t, input), analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Profile.Name != "roff" || report.Flags.RequiresEvaluation || !report.Flags.HasSideEffectSyntax || !report.Flags.HasInclude || !report.Flags.HasOpaque {
		t.Fatalf("stored material became live: profile %+v flags %+v", report.Profile, report.Flags)
	}
	for _, finding := range report.Findings {
		if finding.Feature == "side-effect-syntax" || finding.Feature == "conditional" || finding.Feature == "include" || finding.Feature == "escape" && finding.Target == "AFTER_COMMENT" {
			if finding.Active || finding.Capability.Semantics != analyze.NotAssessed {
				t.Fatal("stored/comment material modeled", finding)
			}
		}
		if finding.Feature == "comment" && finding.Capability.Syntax != analyze.Parsed {
			t.Fatal("known comment incorrectly opaque")
		}
	}
}
func TestProfileUncertaintyAndForeignCatalogue(t *testing.T) {
	for _, tc := range []struct{ input, profile, confidence string }{
		{".SH FIRST\n.SH SECOND\n", "unknown", "low"},
		{"ordinary text\n", "unknown", "low"},
		{".TL\nTitle\n.AU\nAuthor\n", "ms", "high"},
		{".TH TOOL 1\n.TL\nTitle\n", "mixed", "high"},
		{".Dd date\n.Dt TOOL 1\n", "mdoc", "high"},
		{".de LOCAL\nbody\n..\n.LOCAL\n", "custom", "medium"},
	} {
		t.Run(tc.profile+tc.input[:1], func(t *testing.T) {
			report, err := analyze.Run(context.Background(), readSet(t, tc.input), analyze.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if report.Profile.Name != tc.profile || report.Profile.Confidence != tc.confidence {
				t.Fatalf("false profile certainty: %+v", report.Profile)
			}
			if tc.profile == "ms" {
				for _, finding := range report.Findings {
					if finding.Rule == "builtin.control" && finding.Name == ".TL" {
						if finding.Capability.Syntax != analyze.Parsed || finding.Package != "ms" || finding.Capability.Semantics != analyze.NotAssessed {
							t.Fatal("parsed foreign-package token became semantic support", finding)
						}
					}
				}
			}
		})
	}
}
func TestResolvedDisabledAndMissingSources(t *testing.T) {
	fsys := fstest.MapFS{"main.1": {Data: []byte(".TH TOOL 1\n.so common\n.so missing\n")}, "common": {Data: []byte("child\n")}}
	for _, disabled := range []bool{false, true} {
		set, err := source.Load(context.Background(), fsys, "main.1", source.Options{NoIncludes: disabled})
		if err != nil {
			t.Fatal(err)
		}
		report, err := analyze.Run(context.Background(), set, analyze.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if !report.Flags.HasInclude || len(report.Includes) != 2 {
			t.Fatal("include instructions disappeared")
		}
		states := map[string]string{}
		for _, finding := range report.Findings {
			if finding.Feature == "include" {
				states[finding.Target] = finding.Capability.Resolution
			}
		}
		if disabled {
			if states["common"] != "disabled" || len(report.Sources) != 1 {
				t.Fatal("disabled resolution ignored")
			}
		} else {
			if states["common"] != "resolved" || states["missing"] != "missing" || len(report.Sources) != 2 {
				t.Fatal("loading was conflated with recognition", states)
			}
		}
	}
}
func TestAnalysisBoundsAndPartialInput(t *testing.T) {
	set := readSet(t, ".TH TOOL 1\n.SH NAME\ntext\n")
	for _, o := range []analyze.Options{{MaxNodes: 1}, {MaxFindings: 1}, {MaxWork: 1}} {
		report, err := analyze.Run(context.Background(), set, o)
		if !errors.Is(err, analyze.ErrLimit) || report.Complete || report.Error == "" {
			t.Fatal("analysis limit not evidenced", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := analyze.Run(ctx, set, analyze.Options{})
	if !errors.Is(err, context.Canceled) || report.Complete {
		t.Fatal("cancellation ignored", err)
	}
	partial, err := source.Read(context.Background(), strings.NewReader(".SH NAME\nlong text\n"), "partial.1", source.Options{MaxBytes: 12})
	if !errors.Is(err, source.ErrLimit) {
		t.Fatal(err)
	}
	report, err = analyze.Run(context.Background(), partial, analyze.Options{})
	if err != nil || !report.Complete || !report.Flags.HasPartialSource || report.Sources[0].Complete {
		t.Fatal("EOF and enumeration completion conflated", err)
	}
	for _, input := range []string{".SH NAME\r\n雪\r\n", "invalid \xff\n", ".de SAVED\nbody\n"} {
		report, err := analyze.Run(context.Background(), readSet(t, input), analyze.Options{})
		if err != nil || len(report.Findings) == 0 {
			t.Fatal("malformed/opaque material dropped", err)
		}
	}
}

func TestRecognizedMalformedAndUnsupportedEscapes(t *testing.T) {
	report, err := analyze.Run(context.Background(), readSet(t, ".TH\n.SH NAME\ntext \\0 \\f[p] \\t\n.ft UNKNOWN\n"), analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var title, space, font, tab, fontRequest bool
	for _, finding := range report.Findings {
		if finding.Feature == "control" && finding.Name == ".TH" {
			title = true
			if finding.Capability.Syntax != analyze.Recognized {
				t.Fatal("malformed title was marked parsed", finding)
			}
		}
		if finding.Feature == "control" && finding.Name == ".ft" {
			fontRequest = true
			if finding.Capability.Semantics == analyze.Modeled {
				t.Fatal("unhandled font request was marked fully modeled")
			}
		}
		if finding.Feature == "escape" {
			switch finding.Target {
			case "0":
				space = true
				if finding.Capability.Semantics != analyze.Unsupported {
					t.Fatal("generic groff escape became supported man decoding", finding)
				}
			case "p":
				font = true
				if finding.Capability.Semantics != analyze.Unsupported {
					t.Fatal("case-sensitive previous-font escape overclaimed", finding)
				}
			case "t":
				tab = true
				if finding.Capability.Semantics != analyze.Modeled {
					t.Fatal("supported tab escape missed", finding)
				}
			}
		}
	}
	if !title || !space || !font || !tab || !fontRequest {
		t.Fatal("negative capability evidence absent")
	}
}

func TestRecoveryMetricsCountPhysicalLines(t *testing.T) {
	input := ".SH NAME\none\ntwo\nthree"
	set, err := source.Read(context.Background(), strings.NewReader(input), "recovery.1", source.Options{MaxNodes: 8})
	if err == nil {
		t.Fatal("expected parser node budget failure")
	}
	report, err := analyze.Run(context.Background(), set, analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Sources[0].Lines != 4 || report.Sources[0].Bytes != len(input) || !report.Flags.HasOpaque {
		t.Fatal("recovery tail became a false physical-line metric", report.Sources[0], report.Flags)
	}
}

func TestPartialCountsDescribeEmittedPrefix(t *testing.T) {
	report, err := analyze.Run(context.Background(), readSet(t, ".TH TOOL 1\n.nr X 1\n"), analyze.Options{MaxFindings: 1})
	if !errors.Is(err, analyze.ErrLimit) || len(report.Findings) != 1 || report.Counts.Controls[".TH"] != 1 || report.Counts.Controls[".nr"] != 0 || report.Flags.HasDefinitions {
		t.Fatal("partial inventory counted an unemitted finding", report.Counts, report.Flags, err)
	}
}

func TestAcceptedStateControlsNeedNoInventedOrigin(t *testing.T) {
	report, err := analyze.Run(context.Background(), readSet(t, ".ft B\n.PD 0\n.SY tool\n.YS\n"), analyze.Options{})
	if err != nil {
		t.Fatal(err)
	}
	controls := 0
	for _, finding := range report.Findings {
		if finding.Feature == "control" {
			controls++
			if finding.Capability.Semantics != analyze.Modeled {
				t.Fatal("accepted state/closing control lost support classification", finding)
			}
		}
	}
	if controls != 4 {
		t.Fatal("state controls absent")
	}
}
