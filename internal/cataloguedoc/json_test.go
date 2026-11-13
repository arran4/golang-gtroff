package cataloguedoc

import "testing"

func TestJSONOwnershipPositions(t *testing.T) {
	raw := []byte(`[
 {"Contents":[
   {"Type":"Macro","Token":".A","DescriptionRef":"later"},
   {"ID":"inline","Description":"Inline\r\ntext"},
   {"DescriptionRef":"inline"}
 ],
 "Description":"Default after Contents"},
 {"DescriptionRef":"inline","Contents":[{}]},
 {"ID":"later","Description":"Later literal","Contents":[{}]}
]`)
	c, err := ParseJSON(raw)
	if err != nil || !c.Extended {
		t.Fatalf("parse: %v", err)
	}
	for _, tc := range []struct {
		group, item, ownerGroup, ownerItem, line int
		text, pointer                            string
	}{
		{0, 0, 2, -1, 9, "Later literal", "/2/Description"},
		{0, 1, 0, 1, 4, "Inline\r\ntext", "/0/Contents/1/Description"},
		{0, 2, 0, 1, 4, "Inline\r\ntext", "/0/Contents/1/Description"},
		{1, 0, 0, 1, 4, "Inline\r\ntext", "/0/Contents/1/Description"},
		{2, 0, 2, -1, 9, "Later literal", "/2/Description"},
	} {
		text, owner, err := c.Resolve(tc.group, tc.item)
		if err != nil || text != tc.text || owner.Group != tc.ownerGroup || owner.Item != tc.ownerItem || owner.Line != tc.line || owner.Pointer != tc.pointer {
			t.Fatalf("%d/%d: text=%q owner=%+v error=%v", tc.group, tc.item, text, owner, err)
		}
	}
}

func TestJSONLegacyAndInvalid(t *testing.T) {
	c, err := ParseJSON([]byte(`[
 {"Contents":[{}],
 "Description":"A"}]
`))
	if err != nil || c.Extended {
		t.Fatalf("legacy parse: %v", err)
	}
	text, owner, err := c.Resolve(0, 0)
	if err != nil || text != "A" || owner.Line != 3 {
		t.Fatalf("legacy owner wrong: %q %+v %v", text, owner, err)
	}
	for _, raw := range []string{
		`not JSON`, `{}`, `[false]`, `[{"Contents":{}}]`,
		`[{"Description":"A","description":"B","Contents":[{}]}]`,
		`[{"ID":2,"Contents":[{}]}]`,
		`[{"DescriptionRef":"missing","Contents":[{}]}]`,
		`[{"Contents":[{"Description":"A","descriptionref":"x"}]}]`,
	} {
		if _, err := ParseJSON([]byte(raw)); err == nil {
			t.Errorf("invalid documentation accepted: %s", raw)
		}
	}
	if _, _, err := c.Resolve(2, 0); err == nil {
		t.Fatal("invalid group address accepted")
	}
}
