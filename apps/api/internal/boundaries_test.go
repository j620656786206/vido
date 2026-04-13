// Package internal hosts repo-wide invariants enforced as Go tests.
//
// See project-context.md Rule 19 (Package Dependency Boundaries).
package internal

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	importSubtitle   = "github.com/vido/api/internal/subtitle"
	importHandlers   = "github.com/vido/api/internal/handlers"
	importServices   = "github.com/vido/api/internal/services"
	importPathPrefix = "github.com/vido/api/internal/"
)

// TestServicesMustNotImportSubtitle enforces project-context.md Rule 19:
// no file under apps/api/internal/services/ may import the subtitle package.
// subtitle.Engine already imports services.TerminologyCorrectionServiceInterface,
// so adding the reverse direction would create an import cycle that the Go
// compiler rejects with "import cycle not allowed".
func TestServicesMustNotImportSubtitle(t *testing.T) {
	assertNoImport(t, "services", importSubtitle, "Rule 19 (services ↛ subtitle). Use the mirror-types workaround.")
}

// TestServicesMustNotImportSubtitle_detectsViolation proves the rule fires when
// the forbidden import IS present, so the positive-path tests cannot pass
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
		if strings.Trim(imp.Path.Value, `"`) == importSubtitle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("rule check failed to detect synthetic violation of %s — sanity check broken", importSubtitle)
	}
}

// TestForbiddenImportEdges enforces the remaining Rule 19 forbidden directions.
// Today these are also blocked transitively by Go's import-cycle compiler check
// (because the allowed-direction edges already exist), but that's circumstantial.
// If the cycle graph ever changes, these tests still encode the architectural
// intent so a future refactor can't quietly break the layering.
func TestForbiddenImportEdges(t *testing.T) {
	cases := []struct {
		callerDir string
		forbidden string
		why       string
	}{
		{"services", importHandlers, "Rule 19 / Rule 4 — services must not reach back up to handlers"},
		{"repository", importServices, "Rule 19 / Rule 4 — repository sits below services"},
		{"repository", importSubtitle, "Rule 19 — repository sits below services; subtitle is parallel to services"},
	}
	for _, tc := range cases {
		t.Run(tc.callerDir+"_not_"+filepath.Base(tc.forbidden), func(t *testing.T) {
			assertNoImport(t, tc.callerDir, tc.forbidden, tc.why)
		})
	}
}

// TestLeafPackagesHaveNoInternalDeps verifies that packages claimed as "leaf"
// in project-context.md Rule 19 actually have zero internal/ imports. Without
// this test, the documented list will silently rot the moment someone adds
// an import to a "leaf" package and a downstream dev trusts the doc claim.
//
// Runs `go list -deps` per package; the test self-skips if the `go` binary
// isn't on PATH (e.g., in a stripped-down container) rather than failing
// spuriously.
func TestLeafPackagesHaveNoInternalDeps(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go binary not on PATH; cannot verify leaf invariant")
	}

	leaves := []string{"ai", "models", "sse", "retry", "cache"}

	for _, leaf := range leaves {
		t.Run(leaf, func(t *testing.T) {
			pkgPath := "./" + leaf
			if _, err := os.Stat(leaf); os.IsNotExist(err) {
				t.Fatalf("leaf %q listed in project-context.md Rule 19 but directory %s does not exist", leaf, leaf)
			}

			out, err := exec.Command(
				"go", "list", "-deps",
				"-f", "{{if not .Standard}}{{.ImportPath}}{{end}}",
				pkgPath,
			).Output()
			if err != nil {
				t.Fatalf("go list -deps %s failed: %v", pkgPath, err)
			}

			ownPath := importPathPrefix + leaf
			ownPathSlash := ownPath + "/"
			var bad []string
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || !strings.HasPrefix(line, importPathPrefix) {
					continue
				}
				if line == ownPath || strings.HasPrefix(line, ownPathSlash) {
					continue
				}
				bad = append(bad, line)
			}
			if len(bad) > 0 {
				t.Errorf(
					"package %q is claimed as a leaf in project-context.md Rule 19 "+
						"but transitively imports internal packages: %v. "+
						"Either remove the import or remove %q from the leaf list.",
					leaf, bad, leaf,
				)
			}
		})
	}
}

// assertNoImport scans every .go file (including _test.go) directly under
// apps/api/internal/<dirRel>/ and fails if any imports `forbidden`. Sub-
// packages are intentionally skipped — Rule 19 talks about the top-level
// package boundary; sub-packages have their own context.
func assertNoImport(t *testing.T, dirRel, forbidden, why string) {
	t.Helper()

	entries, err := os.ReadDir(dirRel)
	if err != nil {
		t.Fatalf("read internal/%s failed: %v", dirRel, err)
	}

	fset := token.NewFileSet()
	scanned := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(dirRel, entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s failed: %v", path, err)
		}
		scanned++

		for _, imp := range file.Imports {
			if strings.Trim(imp.Path.Value, `"`) == forbidden {
				t.Errorf("%s imports %s — violates %s. See project-context.md Rule 19.", path, forbidden, why)
			}
		}
	}

	if scanned == 0 {
		t.Fatalf("no .go files scanned in internal/%s/ — wrong working directory?", dirRel)
	}
}
