package groff

import (
	"errors"
	"fmt"
	"strings"
)

// ErrArgumentEncoding reports a value the current lexical profile cannot encode
// as one physical-line argument without changing its decoded value.
var ErrArgumentEncoding = errors.New("argument cannot be encoded faithfully")

// ArgumentEncodingError identifies a zero-based value index, without embedding
// potentially large argument data in the error. Reason describes the limitation.
type ArgumentEncodingError struct {
	Index  int
	Reason string
}

func (e *ArgumentEncodingError) Error() string {
	return fmt.Sprintf("%v at index %d: %s", ErrArgumentEncoding, e.Index, e.Reason)
}
func (e *ArgumentEncodingError) Unwrap() error { return ErrArgumentEncoding }

// CanonicalArguments encodes values for this package's argument lexer. Spacing
// is one ASCII space; comments/source spelling are not retained. It verifies
// every field rather than claiming legacy Quote can represent arbitrary strings.
// CR/LF and unrepresentable escape combinations return ErrArgumentEncoding.
func CanonicalArguments(values []string) (string, error) {
	encoded := make([]string, len(values))
	for i, value := range values {
		if strings.ContainsAny(value, "\r\n") {
			return "", &ArgumentEncodingError{Index: i, Reason: "physical line endings need document-level encoding"}
		}
		candidate := Quote(value)
		if !argumentEncodingMatches(candidate, value) {
			// A quoted value ending in an odd backslash cannot close its quote.
			// Unquoted escaped spaces can encode some such values faithfully.
			candidate = strings.ReplaceAll(value, " ", `\ `)
			if !argumentEncodingMatches(candidate, value) {
				return "", &ArgumentEncodingError{Index: i, Reason: "quoting would alter escape or argument boundaries"}
			}
		}
		encoded[i] = candidate
	}
	joined := strings.Join(encoded, " ")
	parsed, err := scanArguments(joined, captureArgumentValues)
	for i, value := range values {
		if err != nil || parsed.hasComment || i >= len(parsed.values) || parsed.values[i] != value {
			return "", &ArgumentEncodingError{Index: i, Reason: "a following separator changes escape or argument boundaries"}
		}
	}
	if len(parsed.values) != len(values) {
		return "", &ArgumentEncodingError{Index: len(values) - 1, Reason: "serialization introduces an extra argument"}
	}
	return joined, nil
}
func argumentEncodingMatches(source, value string) bool {
	list, err := scanArguments(source, captureArgumentValues)
	return err == nil && !list.hasComment && len(list.values) == 1 && list.values[0] == value
}

// Canonical encodes only complete decoded arguments, omitting comments and
// original whitespace. On syntax failure it returns the original cause; use
// Source to retain the malformed tail instead of silently serializing a prefix.
func (a *ArgumentList) Canonical() (string, error) {
	if a.err != nil {
		return "", a.err
	}
	values := make([]string, len(a.lexemes))
	for i, lexeme := range a.lexemes {
		values[i] = lexeme.Value
	}
	return CanonicalArguments(values)
}
