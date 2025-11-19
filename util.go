package groff

import (
	"fmt"
	"strings"
)

func ForwardMatcher(args []string, optional ...bool) ([]string, error) {
	result := make([]string, len(optional))
	for i, opt := range optional {
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
	for i := len(optional) - 1; i >= 0; i-- {
		opt := optional[i]
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

func Quote(s string) string {
	if strings.ContainsAny(s, " \"\t\n") || s == "" {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
