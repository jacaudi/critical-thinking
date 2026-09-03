package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

// TestPrintSchema is relocated from main_test.go; printSchema now lives in schema.go.
func TestPrintSchema(t *testing.T) {
	var buf bytes.Buffer
	if err := printSchema(&buf); err != nil {
		t.Fatalf("printSchema: %v", err)
	}
	got := buf.String()
	for _, want := range []string{`"name": "criticalthinking"`, `"inputSchema"`, `"outputSchema"`} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("schema output missing %q\ngot: %s", want, got)
		}
	}
}

// TestSchemaCmdMatchesPrintSchema asserts the subcommand emits exactly what
// printSchema writes (pin: schema output matches printSchema).
func TestSchemaCmdMatchesPrintSchema(t *testing.T) {
	var want bytes.Buffer
	if err := printSchema(&want); err != nil {
		t.Fatalf("printSchema: %v", err)
	}

	cmd := newSchemaCmd()
	var got bytes.Buffer
	cmd.SetOut(&got)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.String() != want.String() {
		t.Errorf("schema subcommand output != printSchema output\ngot:  %s\nwant: %s", got.String(), want.String())
	}
}

// TestInputSchemaShape pins what MCP clients receive in tools/list: exactly
// the required set, no unknown properties, and nullable unions only where the
// Go type forces them (a pointer and a slice).
func TestInputSchemaShape(t *testing.T) {
	var out bytes.Buffer
	if err := printSchema(&out); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		InputSchema struct {
			Required             []string                   `json:"required"`
			AdditionalProperties *bool                      `json:"additionalProperties"`
			Properties           map[string]json.RawMessage `json:"properties"`
		} `json:"inputSchema"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("schema output is not the expected JSON: %v\n%s", err, out.String())
	}
	wantRequired := []string{"thought", "thoughtNumber", "totalThoughts", "nextThoughtNeeded", "confidence", "assumptions", "critique", "counterArgument"}
	slices.Sort(wantRequired)
	got := slices.Clone(doc.InputSchema.Required)
	slices.Sort(got)
	if !slices.Equal(got, wantRequired) {
		t.Errorf("required = %v, want %v", got, wantRequired)
	}
	if doc.InputSchema.AdditionalProperties == nil || *doc.InputSchema.AdditionalProperties {
		t.Errorf("additionalProperties must be present and false (unknown inputs are rejected)")
	}
	nullable := map[string]bool{"nextThoughtNeeded": true, "assumptions": true}
	for name, raw := range doc.InputSchema.Properties {
		hasNull := bytes.Contains(raw, []byte(`"null"`))
		if hasNull != nullable[name] {
			t.Errorf("property %s nullable=%v, want %v: %s", name, hasNull, nullable[name], raw)
		}
	}
	for _, name := range []string{"isRevision", "revisesThought", "branchFromThought", "branchId", "episodeId", "needsMoreThoughts"} {
		if _, ok := doc.InputSchema.Properties[name]; !ok {
			t.Errorf("property %s missing from schema", name)
		}
	}
	// The deprecated inputs stay in the schema (unknown properties are
	// rejected) but must tell the model they are ignored.
	for _, name := range []string{"episodeId", "needsMoreThoughts"} {
		if !bytes.Contains(doc.InputSchema.Properties[name], []byte(`Deprecated and ignored`)) {
			t.Errorf("property %s must carry the deprecation description: %s", name, doc.InputSchema.Properties[name])
		}
	}
}
