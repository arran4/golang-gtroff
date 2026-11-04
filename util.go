package groff

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnexpectedArguments reports arguments supplied to a zero-argument binder.
var ErrUnexpectedArguments = errors.New("unexpected arguments")

// UnexpectedArgumentsError reports an actual argument count and the allowed
// maximum. Nonzero legacy matcher slots still retain their historical joining.
type UnexpectedArgumentsError struct{ Actual, Maximum int }

func (e *UnexpectedArgumentsError) Error() string {
	return fmt.Sprintf("%v: got %d, maximum %d", ErrUnexpectedArguments, e.Actual, e.Maximum)
}
func (e *UnexpectedArgumentsError) Unwrap() error { return ErrUnexpectedArguments }

func ForwardMatcher(args []string, optional ...bool) ([]string, error) {
	if len(optional) == 0 && len(args) > 0 {
		return nil, &UnexpectedArgumentsError{Actual: len(args), Maximum: 0}
	}
	result := make([]string, len(optional))
	required := requiredArguments(optional)
	for i, opt := range optional {
		if !opt {
			required--
		} else if len(args) <= required {
			continue // Reserve arguments for the required slots still to bind.
		}
		if i == len(optional)-1 {
			// Last slot, consume all remaining args
			if len(args) > 0 {
				result[i] = strings.Join(args, " ")
				args = nil
			} else if !opt {
				return nil, fmt.Errorf("missing required argument %d", i+1)
			}
		} else {
			if len(args) > 0 {
				result[i] = args[0]
				args = args[1:]
			} else if !opt {
				return nil, fmt.Errorf("missing required argument %d", i+1)
			}
		}
	}
	return result, nil
}

func BackwardMatcher(args []string, optional ...bool) ([]string, error) {
	if len(optional) == 0 && len(args) > 0 {
		return nil, &UnexpectedArgumentsError{Actual: len(args), Maximum: 0}
	}
	result := make([]string, len(optional))
	required := requiredArguments(optional)
	for i := len(optional) - 1; i >= 0; i-- {
		opt := optional[i]
		if !opt {
			required--
		} else if len(args) <= required {
			continue
		}
		if i == 0 {
			// First slot, consume all remaining args
			if len(args) > 0 {
				result[i] = strings.Join(args, " ")
				args = nil
			} else if !opt {
				return nil, fmt.Errorf("missing required argument %d", i+1)
			}
		} else {
			if len(args) > 0 {
				result[i] = args[len(args)-1]
				args = args[:len(args)-1]
			} else if !opt {
				return nil, fmt.Errorf("missing required argument %d", i+1)
			}
		}
	}
	return result, nil
}

func requiredArguments(optional []bool) int {
	n := 0
	for _, opt := range optional {
		if !opt {
			n++
		}
	}
	return n
}

// Quote retains legacy argument formatting. CanonicalArguments provides checked
// serialization when decoded values and argument boundaries must round trip.
func Quote(s string) string {
	if strings.ContainsAny(s, " \"\t\n") || s == "" {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
