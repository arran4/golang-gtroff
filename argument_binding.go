package groff

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ArgumentMode identifies an explicit parser/serializer policy.
type ArgumentMode string

const (
	ArgumentsFixed       ArgumentMode = "fixed"
	ArgumentsJoined      ArgumentMode = "joined"
	ArgumentsVariadic    ArgumentMode = "variadic"
	ArgumentsRaw         ArgumentMode = "raw"
	ArgumentsString      ArgumentMode = "string"
	ArgumentsRegister    ArgumentMode = "register"
	ArgumentsConditional ArgumentMode = "conditional"
	ArgumentsDelimited   ArgumentMode = "delimited"
)

// ArgumentGrammar describes typed-field binding, independently of source spelling.
// Optional has one entry per existing typed field. Rest identifies the final
// aggregate or raw-tail field. Minimum counts grammar arguments before binding;
// RepeatFrom/Step describe a variadic prefix and complete repeating groups.
// Flags are optional literal-valued slots. Numeric marks fixed fields whose
// balanced parenthesized expressions may contain spaces; it does not evaluate.
type ArgumentGrammar struct {
	Mode       ArgumentMode   `json:"mode"`
	Optional   []bool         `json:"optional"`
	Reverse    bool           `json:"reverse,omitempty"`
	Rest       int            `json:"rest,omitempty"`
	Minimum    int            `json:"minimum"`
	RepeatFrom int            `json:"repeat_from,omitempty"`
	Step       int            `json:"step,omitempty"`
	Flags      map[int]string `json:"flags,omitempty"`
	Numeric    []bool         `json:"numeric,omitempty"`
}

var (
	ErrMissingArguments = errors.New("missing required arguments")
	ErrInvalidArguments = errors.New("invalid argument syntax")
	ErrArgumentGrammar  = errors.New("invalid argument grammar")
)

type MissingArgumentsError struct{ Actual, Minimum int }

func (e *MissingArgumentsError) Error() string {
	return fmt.Sprintf("%v: got %d, minimum %d", ErrMissingArguments, e.Actual, e.Minimum)
}
func (e *MissingArgumentsError) Unwrap() error { return ErrMissingArguments }

type ArgumentBindingError struct {
	Offset int
	Reason string
}

func (e *ArgumentBindingError) Error() string {
	return fmt.Sprintf("%v at byte %d: %s", ErrInvalidArguments, e.Offset, e.Reason)
}
func (e *ArgumentBindingError) Unwrap() error { return ErrInvalidArguments }

func (g ArgumentGrammar) validate() error {
	n := len(g.Optional)
	if g.Minimum < 0 || g.Step < 0 || g.RepeatFrom < 0 {
		return fmt.Errorf("%w: negative count", ErrArgumentGrammar)
	}
	if g.Minimum < requiredArguments(g.Optional) {
		return fmt.Errorf("%w: minimum omits required fields", ErrArgumentGrammar)
	}
	switch g.Mode {
	case ArgumentsFixed:
		if g.Minimum > n {
			return fmt.Errorf("%w: fixed minimum exceeds fields", ErrArgumentGrammar)
		}
	case ArgumentsJoined, ArgumentsVariadic:
		if n == 0 || g.Rest != n-1 || g.Reverse {
			return fmt.Errorf("%w: aggregate must be the last forward field", ErrArgumentGrammar)
		}
		if g.RepeatFrom > n || g.Mode == ArgumentsVariadic && g.Minimum < g.RepeatFrom {
			return fmt.Errorf("%w: repeat prefix exceeds fields", ErrArgumentGrammar)
		}
	case ArgumentsRaw, ArgumentsString:
		if n == 0 || g.Rest != n-1 || g.Reverse || g.Minimum != requiredArguments(g.Optional) {
			return fmt.Errorf("%w: raw tail must be last", ErrArgumentGrammar)
		}
		for _, optional := range g.Optional[:g.Rest] {
			if optional {
				return fmt.Errorf("%w: raw prefixes must be required", ErrArgumentGrammar)
			}
		}
	case ArgumentsRegister:
		if n != 3 || g.Reverse || g.Optional[0] || g.Optional[1] || !g.Optional[2] || g.Minimum != 2 {
			return fmt.Errorf("%w: register needs three fields", ErrArgumentGrammar)
		}
	case ArgumentsConditional:
		if n != 2 || g.Reverse || g.Optional[0] || g.Minimum != requiredArguments(g.Optional) {
			return fmt.Errorf("%w: conditional needs two fields", ErrArgumentGrammar)
		}
	case ArgumentsDelimited:
		if n != 1 || g.Reverse || g.Optional[0] || g.Minimum != 1 {
			return fmt.Errorf("%w: delimited form needs one field", ErrArgumentGrammar)
		}
	default:
		return fmt.Errorf("%w: unknown mode %q", ErrArgumentGrammar, g.Mode)
	}
	for i, literal := range g.Flags {
		if g.Mode != ArgumentsFixed || i < 0 || i >= n || !g.Optional[i] || literal == "" {
			return fmt.Errorf("%w: invalid optional flag", ErrArgumentGrammar)
		}
	}
	if len(g.Numeric) > 0 && (len(g.Numeric) != n || g.Mode != ArgumentsFixed || g.Reverse) {
		return fmt.Errorf("%w: numeric fields require a forward fixed grammar", ErrArgumentGrammar)
	}
	if g.Mode != ArgumentsVariadic && (g.Step != 0 || g.RepeatFrom != 0) {
		return fmt.Errorf("%w: repetition controls require variadic grammar", ErrArgumentGrammar)
	}
	if len(g.Numeric) > 0 {
		optionalSeen := false
		for _, optional := range g.Optional {
			if optional {
				optionalSeen = true
			} else if optionalSeen {
				return fmt.Errorf("%w: numeric grammar requires trailing optional fields", ErrArgumentGrammar)
			}
		}
	}
	return nil
}

// BindArguments applies a declared grammar without evaluating escapes or input.
// Fixed forms reject extras; aggregate forms are explicit, never inferred.
func BindArguments(source string, g ArgumentGrammar) ([]string, error) {
	if err := g.validate(); err != nil {
		return nil, err
	}
	switch g.Mode {
	case ArgumentsRaw, ArgumentsString:
		return bindRawArguments(source, g)
	case ArgumentsRegister:
		return bindRegisterArguments(source)
	case ArgumentsConditional:
		fields, err := bindConditionalArguments(source)
		if err == nil && fields[1] == "" && !g.Optional[1] {
			return nil, &MissingArgumentsError{Actual: 1, Minimum: 2}
		}
		return fields, err
	case ArgumentsDelimited:
		return bindDelimitedArguments(source)
	}
	if len(g.Numeric) > 0 {
		return bindNumericFields(source, g)
	}
	args, _, err := ArgumentParser(source)
	if err != nil {
		return nil, err
	}
	if len(args) < g.Minimum {
		return nil, &MissingArgumentsError{Actual: len(args), Minimum: g.Minimum}
	}
	if g.Mode == ArgumentsFixed && len(args) > len(g.Optional) {
		return nil, &UnexpectedArgumentsError{Actual: len(args), Maximum: len(g.Optional)}
	}
	if g.Mode == ArgumentsVariadic && g.Step > 0 && (len(args)-g.RepeatFrom)%g.Step != 0 {
		return nil, &ArgumentBindingError{Offset: 0, Reason: "incomplete repeating argument group"}
	}
	out := make([]string, len(g.Optional))
	required := requiredArguments(g.Optional)
	for step := 0; step < len(g.Optional); step++ {
		i := step
		if g.Reverse {
			i = len(g.Optional) - 1 - step
		}
		optional := g.Optional[i]
		if !optional {
			required--
		} else if len(args) <= required {
			continue
		}
		if literal, flag := g.Flags[i]; flag && len(args) > 0 {
			index := 0
			if g.Reverse {
				index = len(args) - 1
			}
			if args[index] != literal {
				continue
			}
		}
		if len(args) == 0 {
			if !optional {
				return nil, &MissingArgumentsError{Actual: 0, Minimum: 1}
			}
			continue
		}
		if g.Mode != ArgumentsFixed && i == g.Rest {
			out[i] = strings.Join(args, " ")
			args = nil
			continue
		}
		if g.Reverse {
			out[i] = args[len(args)-1]
			args = args[:len(args)-1]
		} else {
			out[i] = args[0]
			args = args[1:]
		}
	}
	if len(args) > 0 {
		return nil, &ArgumentBindingError{Offset: 0, Reason: "argument does not match an optional literal flag"}
	}
	return out, nil
}

// FormatArguments emits canonical grammar syntax and checks typed-field equality.
// It preserves intentional empty slots; raw tails are emitted as raw syntax.
func FormatArguments(values []string, g ArgumentGrammar) (string, error) {
	if err := g.validate(); err != nil {
		return "", err
	}
	if len(values) != len(g.Optional) {
		return "", fmt.Errorf("%w: field count mismatch", ErrArgumentGrammar)
	}
	count := len(values)
	keepSlots := false
	optionalSeen := false
	for _, optional := range g.Optional {
		if optional {
			optionalSeen = true
		} else if optionalSeen {
			keepSlots = true
		}
	}
	if !keepSlots {
		for count > 0 && g.Optional[count-1] && values[count-1] == "" {
			count--
		}
	}
	fields := make([]string, 0, count)
	for i, value := range values[:count] {
		if literal, flag := g.Flags[i]; flag {
			if value == "" {
				continue
			}
			if value != literal {
				return "", &ArgumentEncodingError{Index: i, Reason: "value does not match the declared flag"}
			}
		}
		fields = append(fields, value)
	}
	var source string
	var err error
	switch g.Mode {
	case ArgumentsConditional:
		prefix, e := CanonicalArguments(values[:1])
		if e != nil {
			return "", e
		}
		source = prefix
		if values[1] != "" {
			if strings.ContainsAny(values[1], "\r\n") {
				return "", &ArgumentEncodingError{Index: 1, Reason: "conditional body contains line endings"}
			}
			source += " " + values[1]
		}
	case ArgumentsRaw, ArgumentsString:
		prefixCount := g.Rest
		if count <= prefixCount {
			source, err = CanonicalArguments(fields)
		} else {
			prefix, e := CanonicalArguments(fields[:prefixCount])
			if e != nil {
				return "", e
			}
			tail := fields[prefixCount]
			if strings.ContainsAny(tail, "\r\n") {
				return "", &ArgumentEncodingError{Index: prefixCount, Reason: "raw tail contains a physical line ending"}
			}
			if g.Mode == ArgumentsString && (strings.HasPrefix(tail, " ") || strings.HasPrefix(tail, "\t") || strings.HasPrefix(tail, `"`)) {
				tail = `"` + tail
			}
			source = prefix
			if source != "" {
				source += " "
			}
			source += tail
		}
	case ArgumentsDelimited:
		source = values[0]
	default:
		source, err = CanonicalArguments(fields)
	}
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(source, "\r\n") {
		return "", &ArgumentEncodingError{Index: 0, Reason: "canonical source contains a physical line ending"}
	}
	parsed, err := BindArguments(source, g)
	if (err != nil || !reflect.DeepEqual(parsed, values)) && g.Mode == ArgumentsVariadic && values[g.Rest] != "" {
		aggregate, _, e := ArgumentParser(values[g.Rest])
		if e == nil && strings.Join(aggregate, " ") == values[g.Rest] {
			expanded := append([]string(nil), values[:g.Rest]...)
			expanded = append(expanded, aggregate...)
			if alternative, e := CanonicalArguments(expanded); e == nil {
				if rebound, e := BindArguments(alternative, g); e == nil && reflect.DeepEqual(rebound, values) {
					return alternative, nil
				}
			}
		}
	}
	if err != nil || !reflect.DeepEqual(parsed, values) {
		return "", &ArgumentEncodingError{Index: 0, Reason: "canonical grammar syntax changes typed fields"}
	}
	return source, nil
}

// CanonicalFormatter is implemented by generated typed tokens. Source snapshots
// remain independent; this formats current fields and can reject ambiguous data.
type CanonicalFormatter interface{ CanonicalString() (string, error) }

// CanonicalToken formats a generated token with its declared grammar.
func CanonicalToken(token Token) (string, error) {
	if nilInterface(token) {
		return "", fmt.Errorf("%w: nil token", ErrArgumentEncoding)
	}
	formatter, ok := token.(CanonicalFormatter)
	if !ok {
		return "", fmt.Errorf("%w: token has no declared serializer", ErrArgumentEncoding)
	}
	return formatter.CanonicalString()
}
