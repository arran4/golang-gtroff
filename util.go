package groff

import (
	"fmt"
	"strings"
)

func ForwardMatcher(args []string, optional ...bool) ([]string, error) {
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

func Quote(s string) string {
	if strings.ContainsAny(s, " \"\t\n") || s == "" {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
