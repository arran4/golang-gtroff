package groff

import (
	"errors"
	"strings"
	"testing"
)

func TestPackageCollisionResolution(t *testing.T) {
	// Test .SH in man vs ms
	tokManSH, err := ParseLineWithPackage(`.SH "Section Title"`, "man")
	if err != nil {
		t.Fatalf("ParseLineWithPackage man .SH error: %v", err)
	}
	manSH, ok := tokManSH.(*TokenManSH)
	if !ok {
		t.Fatalf("expected *TokenManSH, got %T", tokManSH)
	}
	if manSH.Heading != "Section Title" {
		t.Errorf("expected Heading='Section Title', got %q", manSH.Heading)
	}

	tokMsSH, err := ParseLineWithPackage(`.SH 2`, "ms")
	if err != nil {
		t.Fatalf("ParseLineWithPackage ms .SH error: %v", err)
	}
	msSH, ok := tokMsSH.(*TokenMsSH)
	if !ok {
		t.Fatalf("expected *TokenMsSH, got %T", tokMsSH)
	}
	if msSH.Matchlevel != "2" {
		t.Errorf("expected Matchlevel='2', got %q", msSH.Matchlevel)
	}

	// Test .B in man vs ms
	tokManB, err := ParseLineWithPackage(`.B "Bold Text"`, "man")
	if err != nil {
		t.Fatalf("ParseLineWithPackage man .B error: %v", err)
	}
	manB, ok := tokManB.(*TokenManB)
	if !ok {
		t.Fatalf("expected *TokenManB, got %T", tokManB)
	}
	if manB.Text != "Bold Text" {
		t.Errorf("expected Text='Bold Text', got %q", manB.Text)
	}

	tokMsB, err := ParseLineWithPackage(`.B "Bold" ")" "("`, "ms")
	if err != nil {
		t.Fatalf("ParseLineWithPackage ms .B error: %v", err)
	}
	msB, ok := tokMsB.(*TokenMsB)
	if !ok {
		t.Fatalf("expected *TokenMsB, got %T", tokMsB)
	}
	if msB.Txt != "Bold" || msB.Post != ")" || msB.Pre != "(" {
		t.Errorf("expected Txt='Bold' Post=')' Pre='(', got Txt=%q Post=%q Pre=%q", msB.Txt, msB.Post, msB.Pre)
	}

	// Test .BI in man vs ms
	tokMsBI, err := ParseLineWithPackage(`.BI "BoldItalic" ")" "("`, "ms")
	if err != nil {
		t.Fatalf("ParseLineWithPackage ms .BI error: %v", err)
	}
	msBI, ok := tokMsBI.(*TokenMsBI)
	if !ok {
		t.Fatalf("expected *TokenMsBI, got %T", tokMsBI)
	}
	if msBI.Txt != "BoldItalic" || msBI.Post != ")" || msBI.Pre != "(" {
		t.Errorf("expected Txt='BoldItalic' Post=')' Pre='(', got Txt=%q Post=%q Pre=%q", msBI.Txt, msBI.Post, msBI.Pre)
	}

	// Test .NH in ms
	tokMsNH, err := ParseLineWithPackage(`.NH 1`, "ms")
	if err != nil {
		t.Fatalf("ParseLineWithPackage ms .NH error: %v", err)
	}
	msNH, ok := tokMsNH.(*TokenMsNH)
	if !ok {
		t.Fatalf("expected *TokenMsNH, got %T", tokMsNH)
	}
	if msNH.Level != "1" {
		t.Errorf("expected Level='1', got %q", msNH.Level)
	}

	// Test .DS in ms
	tokMsDS, err := ParseLineWithPackage(`.DS C`, "ms")
	if err != nil {
		t.Fatalf("ParseLineWithPackage ms .DS error: %v", err)
	}
	msDS, ok := tokMsDS.(*TokenMsDS)
	if !ok {
		t.Fatalf("expected *TokenMsDS, got %T", tokMsDS)
	}
	if msDS.Placement != "C" {
		t.Errorf("expected Placement='C', got %q", msDS.Placement)
	}
}

func TestRequestParsing(t *testing.T) {
	// .nf and .fi
	tokNf, err := ParseLine(".nf")
	if err != nil {
		t.Fatalf("ParseLine .nf error: %v", err)
	}
	if _, ok := tokNf.(*TokenRequestNf); !ok {
		t.Errorf("expected *TokenRequestNf, got %T", tokNf)
	}

	tokFi, err := ParseLine(".fi")
	if err != nil {
		t.Fatalf("ParseLine .fi error: %v", err)
	}
	if _, ok := tokFi.(*TokenRequestFi); !ok {
		t.Errorf("expected *TokenRequestFi, got %T", tokFi)
	}

	// .br
	tokBr, err := ParseLine(".br")
	if err != nil {
		t.Fatalf("ParseLine .br error: %v", err)
	}
	if _, ok := tokBr.(*TokenRequestBr); !ok {
		t.Errorf("expected *TokenRequestBr, got %T", tokBr)
	}

	// 'br (no-break request)
	tokNoBreakBr, err := ParseLine("'br")
	if err != nil {
		t.Fatalf("ParseLine 'br error: %v", err)
	}
	if _, ok := tokNoBreakBr.(*TokenRequestBr); !ok {
		t.Errorf("expected *TokenRequestBr for 'br, got %T", tokNoBreakBr)
	}

	// .sp
	tokSp, err := ParseLine(".sp 2")
	if err != nil {
		t.Fatalf("ParseLine .sp error: %v", err)
	}
	sp, ok := tokSp.(*TokenRequestSp)
	if !ok {
		t.Errorf("expected *TokenRequestSp, got %T", tokSp)
	}
	if sp.Distance != "2" {
		t.Errorf("expected Distance='2', got %q", sp.Distance)
	}

	// .nr
	tokNr, err := ParseLine(".nr PO 72 2")
	if err != nil {
		t.Fatalf("ParseLine .nr error: %v", err)
	}
	nr, ok := tokNr.(*TokenRequestNr)
	if !ok {
		t.Errorf("expected *TokenRequestNr, got %T", tokNr)
	}
	if nr.Ident != "PO" || nr.Value != "72" || nr.Incr != "2" {
		t.Errorf("expected Ident='PO' Value='72' Incr='2', got %v", nr)
	}
}

func TestEmptyQuotedArguments(t *testing.T) {
	// Empty quoted string in .TH
	tokTH, err := ParseLine(`.TH TOOL 1 "" "Header Middle" "Footer"`)
	if err != nil {
		t.Fatalf("ParseLine .TH with empty quote error: %v", err)
	}
	th, ok := tokTH.(*TokenMacroTH)
	if !ok {
		t.Fatalf("expected *TokenMacroTH, got %T", tokTH)
	}
	if th.Title != "TOOL" || th.Section != "1" || th.Extra1 != "" || th.Extra2 != "Header Middle" || th.Extra3 != "Footer" {
		t.Errorf("unexpected .TH fields: %+v", th)
	}
}

func TestStrictModeUnknownCommand(t *testing.T) {
	_, err := ParseLine(".UNKNOWNMACRO arg")
	if err == nil {
		t.Errorf("expected error in strict ParseLine for unknown macro, got nil")
	}
	if !errors.Is(err, ErrNoTokenMatch) {
		t.Errorf("expected errors.Is(err, ErrNoTokenMatch), got %v", err)
	}
}

func TestPermissiveModeUnknownCommand(t *testing.T) {
	tok, err := ParseLinePermissive(".UNKNOWNMACRO arg", "")
	if err != nil {
		t.Fatalf("unexpected error in ParseLinePermissive: %v", err)
	}
	unk, ok := tok.(*TokenUnknownControl)
	if !ok {
		t.Fatalf("expected *TokenUnknownControl, got %T", tok)
	}
	if unk.Name != ".UNKNOWNMACRO" || unk.RawArgs != "arg" {
		t.Errorf("unexpected TokenUnknownControl: %+v", unk)
	}
	if unk.String() != ".UNKNOWNMACRO arg" {
		t.Errorf("String() mismatch: got %q, want %q", unk.String(), ".UNKNOWNMACRO arg")
	}
}

func TestRoundTripFidelity(t *testing.T) {
	cases := []string{
		`.TH SIMPLE 1 2024-01-01 Test "Test Manual"`,
		`.SH "SECTION NAME"`,
		`.B bold`,
		`.IP \(bu 2`,
		`.br`,
		`.nf`,
		`.fi`,
	}
	for _, c := range cases {
		tok, err := ParseLine(c)
		if err != nil {
			t.Errorf("ParseLine(%q) error: %v", c, err)
			continue
		}
		serialized := tok.String()
		if serialized != c {
			t.Errorf("Round trip mismatch: got %q, want %q", serialized, c)
		}
	}
}

func TestUnicodeSafety(t *testing.T) {
	unicodeLines := []string{
		`.SH "Section with Unicode: 日本語 and Emojis 🚀"`,
		`.B "café au lait — crème brûlée"`,
	}
	for _, line := range unicodeLines {
		tok, err := ParseLine(line)
		if err != nil {
			t.Errorf("ParseLine(%q) error: %v", line, err)
			continue
		}
		if !strings.Contains(tok.String(), "日本語") && !strings.Contains(tok.String(), "café") {
			t.Errorf("Unicode corruption in tok.String(): %q", tok.String())
		}
	}
}
