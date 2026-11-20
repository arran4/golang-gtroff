package lint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	groff "github.com/arran4/golang-gtroff"
	"github.com/arran4/golang-gtroff/internal/sourceio"
)

// ParsePolicy reads an explicit schema-version-1 JSON policy, limited to 64 KiB.
// It cannot install callbacks, execute scripts or load other files. The reader is
// borrowed. Validate the policy against the selected Registry before source I/O.
func ParsePolicy(reader io.Reader) (Policy, error) {
	if sourceio.IsNil(reader) {
		return Policy{}, fmt.Errorf("%w: nil policy reader", ErrConfig)
	}
	checked := sourceio.NewReader(reader, func(count, size int, err error) error {
		return &groff.ReaderCountError{Count: count, Size: size, Err: err}
	})
	data, err := io.ReadAll(io.LimitReader(checked, (64<<10)+1))
	if err != nil {
		return Policy{}, err
	}
	if len(data) > 64<<10 {
		return Policy{}, fmt.Errorf("%w: policy exceeds 64 KiB", ErrConfig)
	}
	var config struct {
		SchemaVersion int    `json:"schemaVersion"`
		Policy        Policy `json:"policy"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Policy{}, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Policy{}, fmt.Errorf("%w: trailing data", ErrConfig)
	}
	if config.SchemaVersion != 1 {
		return Policy{}, fmt.Errorf("%w: unsupported schemaVersion", ErrConfig)
	}
	return config.Policy, nil
}

// ValidatePolicy rejects unknown rule IDs and invalid settings without reading
// documents. Callers can validate against a registry with trusted custom rules.
func ValidatePolicy(policy Policy, registry *Registry) error {
	if registry == nil {
		var err error
		registry, err = NewRegistry()
		if err != nil {
			return err
		}
	}
	ids := map[string]bool{}
	for _, rule := range registry.rules {
		ids[rule.Info.ID] = true
	}
	if len(policy.RequiredSections) > 64 || len(policy.Enable) > 128 || len(policy.Disable) > 128 || len(policy.Severity) > 128 {
		return fmt.Errorf("%w: policy list budget exceeded", ErrConfig)
	}
	seenSections := map[string]bool{}
	for _, section := range policy.RequiredSections {
		if strings.TrimSpace(section) == "" || seenSections[normalizeHeading(section)] || len(section) > 256 || strings.ContainsAny(section, "\r\n\x00") {
			return fmt.Errorf("%w: invalid required section", ErrConfig)
		}
		seenSections[normalizeHeading(section)] = true
	}
	for _, list := range [][]string{policy.Enable, policy.Disable} {
		for _, id := range list {
			if !ids[id] {
				return fmt.Errorf("%w: unknown rule %q", ErrConfig, id)
			}
		}
	}
	for id, severity := range policy.Severity {
		if !ids[id] || severityRank(severity) == 0 {
			return fmt.Errorf("%w: invalid rule severity %q", ErrConfig, id)
		}
	}
	if policy.MaxIndentDepth < 0 || policy.MaxIndentDepth > 128 {
		return fmt.Errorf("%w: indent depth must be 0..128", ErrConfig)
	}
	return nil
}
