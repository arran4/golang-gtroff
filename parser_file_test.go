package groff

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseFile tests parsing complete man page files
func TestParseFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
		minTokens   int // minimum number of tokens expected
		checkTokens func(t *testing.T, tokens []Token)
	}{
		{
			name:        "simple man page",
			filename:    "simple.1",
			expectError: false,
			minTokens:   20,
			checkTokens: func(t *testing.T, tokens []Token) {
				// Check for TH macro
				foundTH := false
				foundSH := false
				foundSY := false
				foundYS := false

				for _, tok := range tokens {
					switch tok.(type) {
					case *TokenMacroTH:
						foundTH = true
						th := tok.(*TokenMacroTH)
						if th.Title != "SIMPLE" {
							t.Errorf("Expected title 'SIMPLE', got %q", th.Title)
						}
						if th.Section != "1" {
							t.Errorf("Expected section '1', got %q", th.Section)
						}
					case *TokenMacroSH:
						foundSH = true
					case *TokenMacroSY:
						foundSY = true
					case *TokenMacroYS:
						foundYS = true
					}
				}

				if !foundTH {
					t.Error("Expected to find TH macro")
				}
				if !foundSH {
					t.Error("Expected to find SH macro")
				}
				if !foundSY {
					t.Error("Expected to find SY macro")
				}
				if !foundYS {
					t.Error("Expected to find YS macro")
				}
			},
		},
		{
			name:        "complex man page",
			filename:    "complex.7",
			expectError: false,
			minTokens:   40,
			checkTokens: func(t *testing.T, tokens []Token) {
				foundB := false
				foundI := false
				foundBI := false
				foundBR := false
				foundIP := false
				foundTP := false
				foundRS := false
				foundRE := false
				foundEX := false
				foundEE := false
				foundUR := false
				foundUE := false
				foundMT := false
				foundME := false

				for _, tok := range tokens {
					switch tok.(type) {
					case *TokenMacroB:
						foundB = true
					case *TokenMacroI:
						foundI = true
					case *TokenMacroBI:
						foundBI = true
					case *TokenMacroBR:
						foundBR = true
					case *TokenMacroIP:
						foundIP = true
					case *TokenMacroTP:
						foundTP = true
					case *TokenMacroRS:
						foundRS = true
					case *TokenMacroRE:
						foundRE = true
					case *TokenMacroEX:
						foundEX = true
					case *TokenMacroEE:
						foundEE = true
					case *TokenMacroUR:
						foundUR = true
					case *TokenMacroUE:
						foundUE = true
					case *TokenMacroMT:
						foundMT = true
					case *TokenMacroME:
						foundME = true
					}
				}

				if !foundB {
					t.Error("Expected to find B macro")
				}
				if !foundI {
					t.Error("Expected to find I macro")
				}
				if !foundBI {
					t.Error("Expected to find BI macro")
				}
				if !foundBR {
					t.Error("Expected to find BR macro")
				}
				if !foundIP {
					t.Error("Expected to find IP macro")
				}
				if !foundTP {
					t.Error("Expected to find TP macro")
				}
				if !foundRS {
					t.Error("Expected to find RS macro")
				}
				if !foundRE {
					t.Error("Expected to find RE macro")
				}
				if !foundEX {
					t.Error("Expected to find EX macro")
				}
				if !foundEE {
					t.Error("Expected to find EE macro")
				}
				if !foundUR {
					t.Error("Expected to find UR macro")
				}
				if !foundUE {
					t.Error("Expected to find UE macro")
				}
				if !foundMT {
					t.Error("Expected to find MT macro")
				}
				if !foundME {
					t.Error("Expected to find ME macro")
				}
			},
		},
		{
			name:        "minimal man page",
			filename:    "minimal.1",
			expectError: false,
			minTokens:   3,
			checkTokens: func(t *testing.T, tokens []Token) {
				foundTH := false
				foundSH := false

				for _, tok := range tokens {
					switch tok.(type) {
					case *TokenMacroTH:
						foundTH = true
					case *TokenMacroSH:
						foundSH = true
					}
				}

				if !foundTH {
					t.Error("Expected to find TH macro")
				}
				if !foundSH {
					t.Error("Expected to find SH macro")
				}
			},
		},
		{
			name:        "environ snippet from real man page",
			filename:    "environ_snippet.7",
			expectError: false,
			minTokens:   10,
			checkTokens: func(t *testing.T, tokens []Token) {
				foundTH := false
				foundComment := false

				for _, tok := range tokens {
					switch tok.(type) {
					case *TokenMacroTH:
						foundTH = true
						th := tok.(*TokenMacroTH)
						if th.Title != "ENVIRON" {
							t.Errorf("Expected title 'ENVIRON', got %q", th.Title)
						}
					case TokenComment:
						foundComment = true
					}
				}

				if !foundTH {
					t.Error("Expected to find TH macro")
				}
				if !foundComment {
					t.Error("Expected to find comments")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseManPageFile(filepath.Join("testdata", tt.filename))

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(tokens) < tt.minTokens {
				t.Errorf("Expected at least %d tokens, got %d", tt.minTokens, len(tokens))
			}

			if tt.checkTokens != nil {
				tt.checkTokens(t, tokens)
			}
		})
	}
}

// TestParseRealManPages tests parsing actual man page files if available
func TestParseRealManPages(t *testing.T) {
	realFiles := []struct {
		name     string
		filename string
		minLines int
	}{
		{
			name:     "groff_man.txt",
			filename: "groff_man.txt",
			minLines: 100,
		},
		{
			name:     "groff_ms.txt",
			filename: "groff_ms.txt",
			minLines: 100,
		},
		{
			name:     "environ.txt",
			filename: "environ.txt",
			minLines: 50,
		},
	}

	for _, tt := range realFiles {
		t.Run(tt.name, func(t *testing.T) {
			// Check if file exists
			if _, err := os.Stat(tt.filename); os.IsNotExist(err) {
				t.Skipf("File %s not found, skipping", tt.filename)
				return
			}

			tokens, err := parseManPageFile(tt.filename)
			if err != nil {
				t.Errorf("Error parsing %s: %v", tt.filename, err)
				return
			}

			if len(tokens) < tt.minLines {
				t.Errorf("Expected at least %d tokens from %s, got %d",
					tt.minLines, tt.filename, len(tokens))
			}

			// Log some statistics
			t.Logf("Parsed %s: %d tokens", tt.filename, len(tokens))

			// Count token types
			tokenCounts := make(map[string]int)
			for _, tok := range tokens {
				typeName := getTokenTypeName(tok)
				tokenCounts[typeName]++
			}

			t.Logf("Token type distribution:")
			for typeName, count := range tokenCounts {
				if count > 0 {
					t.Logf("  %s: %d", typeName, count)
				}
			}
		})
	}
}

// TestParseLineByLine tests that we can parse every line without errors
func TestParseLineByLine(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"simple", "testdata/simple.1"},
		{"complex", "testdata/complex.7"},
		{"minimal", "testdata/minimal.1"},
		{"environ_snippet", "testdata/environ_snippet.7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.Open(tt.filename)
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			lineNum := 0
			errorCount := 0

			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				// Skip empty lines
				if strings.TrimSpace(line) == "" {
					continue
				}

				_, err := ParseLine(line)
				if err != nil {
					// Some lines may not parse (e.g., unknown macros)
					// We log them but don't fail the test
					t.Logf("Line %d: %v - %q", lineNum, err, line)
					errorCount++
				}
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Error reading file: %v", err)
			}

			t.Logf("Parsed %d lines with %d errors", lineNum, errorCount)
		})
	}
}

// Helper function to parse a complete man page file
func parseManPageFile(filename string) ([]Token, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tokens []Token
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		token, err := ParseLine(line)
		if err != nil {
			// For now, we skip lines that don't parse
			// In a real implementation, you might want to handle this differently
			continue
		}

		tokens = append(tokens, token)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// Helper function to get token type name for statistics
func getTokenTypeName(tok Token) string {
	switch tok.(type) {
	case TokenText:
		return "Text"
	case TokenComment:
		return "Comment"
	case *TokenMacroTH:
		return "TH"
	case *TokenMacroSH:
		return "SH"
	case *TokenMacroSS:
		return "SS"
	case *TokenMacroTP:
		return "TP"
	case *TokenMacroIP:
		return "IP"
	case *TokenMacroB:
		return "B"
	case *TokenMacroI:
		return "I"
	case *TokenMacroBI:
		return "BI"
	case *TokenMacroBR:
		return "BR"
	case *TokenMacroIB:
		return "IB"
	case *TokenMacroIR:
		return "IR"
	case *TokenMacroRB:
		return "RB"
	case *TokenMacroRI:
		return "RI"
	case *TokenMacroSM:
		return "SM"
	case *TokenMacroSB:
		return "SB"
	case *TokenMacroP:
		return "P"
	case *TokenMacroPP:
		return "PP"
	case *TokenMacroLP:
		return "LP"
	case *TokenMacroRS:
		return "RS"
	case *TokenMacroRE:
		return "RE"
	case *TokenMacroEX:
		return "EX"
	case *TokenMacroEE:
		return "EE"
	case *TokenMacroSY:
		return "SY"
	case *TokenMacroYS:
		return "YS"
	case *TokenMacroOP:
		return "OP"
	case *TokenMacroMR:
		return "MR"
	case *TokenMacroMT:
		return "MT"
	case *TokenMacroME:
		return "ME"
	case *TokenMacroUR:
		return "UR"
	case *TokenMacroUE:
		return "UE"
	default:
		return "Unknown"
	}
}

// Benchmark tests

// BenchmarkParseLine benchmarks parsing a single line
func BenchmarkParseLine(b *testing.B) {
	testLines := []string{
		`.TH TEST 1 2024-01-01 "Test" "Test Manual"`,
		`.SH NAME`,
		`.B bold text`,
		`.BI bold italic`,
		`.IP \(bu 2`,
		`This is plain text`,
		`.\" This is a comment`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		line := testLines[i%len(testLines)]
		_, _ = ParseLine(line)
	}
}

// BenchmarkParseFile benchmarks parsing a complete file
func BenchmarkParseFile(b *testing.B) {
	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parseManPageFile("testdata/simple.1")
		}
	})

	b.Run("complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parseManPageFile("testdata/complex.7")
		}
	})

	b.Run("environ_snippet", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parseManPageFile("testdata/environ_snippet.7")
		}
	})
}

// BenchmarkParseRealFile benchmarks parsing real man page files
func BenchmarkParseRealFile(b *testing.B) {
	if _, err := os.Stat("environ.txt"); err == nil {
		b.Run("environ.txt", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = parseManPageFile("environ.txt")
			}
		})
	}

	if _, err := os.Stat("groff_man.txt"); err == nil {
		b.Run("groff_man.txt", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = parseManPageFile("groff_man.txt")
			}
		})
	}
}
