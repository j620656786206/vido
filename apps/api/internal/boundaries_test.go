// Package internal hosts repo-wide invariants enforced as Go tests.
//
// See project-context.md Rule 19 (Package Dependency Boundaries).
package internal

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const forbiddenSubtitleImport = "github.com/vido/api/internal/subtitle"

// TestServicesMustNotImportSubtitle enforces project-context.md Rule 19:
// no file under apps/api/internal/services/ may import the subtitle package.
// subtitle.Engine already imports services.TerminologyCorrectionServiceInterface,
// so adding the reverse direction would create an import cycle that the Go
// compiler rejects with "import cycle not allowed".
//
// Runs from apps/api/internal/ (the package directory) so the relative path
// "services" resolves to apps/api/internal/services/.
func TestServicesMustNotImportSubtitle(t *testing.T) {
	const servicesDir = "services"

	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		t.Fatalf("read internal/services failed: %v", err)
	}

	fset := token.NewFileSet()
	scanned := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(servicesDir, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s failed: %v", path, err)
		}
		scanned++

		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if importPath == forbiddenSubtitleImport {
				t.Errorf(
					"%s imports %s — violates Rule 19 (services ↛ subtitle). "+
						"Use the mirror-types workaround (see project-context.md Rule 19); "+
						"reference implementation: services/translation_service.go TranslationBlock.",
					path, importPath,
				)
			}
		}
	}

	if scanned == 0 {
		t.Fatal("no .go files scanned in internal/services/ — wrong working directory?")
	}
}

// TestServicesMustNotImportSubtitle_detectsViolation proves the rule fires when
// the forbidden import IS present, so the positive-path test cannot pass
// vacuously (e.g., if the file walk silently inspected nothing).
func TestServicesMustNotImportSubtitle_detectsViolation(t *testing.T) {
	const violatingSrc = `package services

import (
	_ "github.com/vido/api/internal/subtitle"
)
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "synthetic.go", violatingSrc, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse synthetic source failed: %v", err)
	}

	found := false
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if importPath == forbiddenSubtitleImport {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("rule check failed to detect synthetic violation of %s — sanity check broken", forbiddenSubtitleImport)
	}
}
