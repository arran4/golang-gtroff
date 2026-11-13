package cataloguedoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Address identifies the original literal, including when a shared description
// lives inside an item. Item is -1 for a group default.
type Address struct {
	Group, Item, Line int
	Pointer           string
}

type Catalogue struct {
	resolver *Resolver
	indices  [][]int
	owners   []Address
	Extended bool
}

type jsonSpan struct {
	raw       json.RawMessage
	start     int
	keyStart  int
	fieldName string
}

func elements(raw []byte, base int) ([]jsonSpan, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	if token, err := d.Token(); err != nil || token != json.Delim('[') {
		return nil, fmt.Errorf("expected catalogue array")
	}
	var result []jsonSpan
	for d.More() {
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, err
		}
		result = append(result, jsonSpan{raw: value, start: base + int(d.InputOffset()) - len(value)})
	}
	_, err := d.Token()
	return result, err
}

func fields(span jsonSpan) (map[string]jsonSpan, error) {
	result := make(map[string]jsonSpan)
	if bytes.Equal(bytes.TrimSpace(span.raw), []byte("null")) {
		return result, nil
	}
	d := json.NewDecoder(bytes.NewReader(span.raw))
	if token, err := d.Token(); err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("expected catalogue object")
	}
	for d.More() {
		keyStart := int(d.InputOffset())
		for keyStart < len(span.raw) && strings.ContainsRune(" ,\t\r\n", rune(span.raw[keyStart])) {
			keyStart++
		}
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("expected catalogue field name")
		}
		canonical := strings.ToLower(key)
		if _, duplicate := result[canonical]; duplicate {
			return nil, fmt.Errorf("duplicate catalogue field %q", key)
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, err
		}
		result[canonical] = jsonSpan{value, span.start + int(d.InputOffset()) - len(value), span.start + keyStart, key}
	}
	_, err := d.Token()
	return result, err
}

func textFields(values map[string]jsonSpan) (Text, error) {
	var t Text
	for _, field := range []struct {
		name string
		out  *string
	}{{"id", &t.ID}, {"description", &t.Literal}, {"descriptionref", &t.Ref}} {
		if value, ok := values[field.name]; ok {
			if err := json.Unmarshal(value.raw, field.out); err != nil {
				return t, fmt.Errorf("invalid documentation %s: %w", field.name, err)
			}
		}
	}
	return t, nil
}

// ParseJSON retains owner positions independently of property order. Unknown
// generator fields are left to the generator's own schema validation.
func ParseJSON(raw []byte) (*Catalogue, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid catalogue JSON")
	}
	groups, err := elements(raw, 0)
	if err != nil {
		return nil, err
	}
	c := &Catalogue{indices: make([][]int, len(groups))}
	var texts []Text
	add := func(t Text, values map[string]jsonSpan, group, item int, pointer string) int {
		index := len(texts)
		texts = append(texts, t)
		address := Address{Group: group, Item: item}
		if field, ok := values["description"]; ok {
			address.Line = bytes.Count(raw[:field.keyStart], []byte{'\n'}) + 1
			address.Pointer = pointer + "/" + field.fieldName
		}
		c.owners = append(c.owners, address)
		return index
	}
	for gi, group := range groups {
		values, err := fields(group)
		if err != nil {
			return nil, err
		}
		doc, err := textFields(values)
		if err != nil {
			return nil, err
		}
		pointer := fmt.Sprintf("/%d", gi)
		parent := add(doc, values, gi, -1, pointer)
		c.Extended = c.Extended || doc.Ref != ""
		contents, ok := values["contents"]
		if !ok {
			continue
		}
		items, err := elements(contents.raw, contents.start)
		if err != nil {
			return nil, err
		}
		for ii, item := range items {
			values, err := fields(item)
			if err != nil {
				return nil, err
			}
			doc, err := textFields(values)
			if err != nil {
				return nil, err
			}
			index := parent
			if doc.ID != "" || doc.Literal != "" || doc.Ref != "" {
				index = add(doc, values, gi, ii, fmt.Sprintf("%s/%s/%d", pointer, contents.fieldName, ii))
				c.Extended = true
			}
			c.indices[gi] = append(c.indices[gi], index)
		}
	}
	c.resolver, err = New(texts)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalogue) Resolve(group, item int) (string, Address, error) {
	if c == nil || group < 0 || group >= len(c.indices) || item < 0 || item >= len(c.indices[group]) {
		return "", Address{}, fmt.Errorf("catalogue documentation address out of range")
	}
	text, owner, err := c.resolver.Resolve(c.indices[group][item])
	if err != nil {
		return "", Address{}, err
	}
	return text, c.owners[owner], nil
}
