package thinking

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestPackageHasNoState is the durable form of design decision D1: the package
// holds no state and imports neither the MCP SDK, OpenTelemetry, nor any
// synchronisation primitive. It parses every non-test file, so a package-level
// var or a new import fails the suite rather than waiting for a code review.
func TestPackageHasNoState(t *testing.T) {
	allowedImports := []string{"encoding/json", "errors", "fmt", "math", "strings"}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no source files found; is the test running in the package directory?")
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !slices.Contains(allowedImports, path) {
				t.Errorf("%s: import %q is not in allowedImports (the package must stay free of SDK, telemetry, and sync; if this import is state-free, add it to the list)", name, path)
			}
		}
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
				t.Errorf("%s:%d: package-level var (the package must hold no state)", name, fset.Position(gd.Pos()).Line)
			}
		}
	}
}
