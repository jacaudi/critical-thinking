package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoProcessState is the adapter-side twin of internal/thinking's
// TestPackageHasNoState: this package once held the session registry, the
// eviction callback, and the idle timer. The only package-level vars allowed
// now are the ldflags build info and the CLI's immutable sentinel error;
// anything else is process state and fails the suite.
func TestNoProcessState(t *testing.T) {
	allowed := map[string]bool{"version": true, "commit": true, "date": true, "errCLIFailed": true}
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
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				for _, n := range spec.(*ast.ValueSpec).Names {
					if !allowed[n.Name] {
						t.Errorf("%s:%d: package-level var %q (the server must hold no process state)", name, fset.Position(n.Pos()).Line, n.Name)
					}
				}
			}
		}
	}
}
