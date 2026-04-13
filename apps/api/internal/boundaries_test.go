// Package internal hosts repo-wide invariants enforced as Go tests.
//
// See project-context.md Rule 19 (Package Dependency Boundaries).
package internal

import (
	"bytes"
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
// no production file under apps/api/internal/services/ may import the
// subtitle package. subtitle.Engine already imports
// services.TerminologyCorrectionServiceInterface, so adding the reverse
// direction would create an import cycle that the Go compiler rejects
// with "import cycle not allowed".
func TestServicesMustNotImportSubtitle(t *testing.T) {
	assertNoImport(t, "services", importSubtitle, "Rule 19 (services ↛ subtitle). Use the mirror-types workaround.")
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
// Executes `go list -deps` per package via the `go` binary that is, by
// definition, running this test.
func TestLeafPackagesHaveNoInternalDeps(t *testing.T) {
	leaves := []string{"ai", "models", "sse", "retry", "cache"}

	for _, leaf := range leaves {
		t.Run(leaf, func(t *testing.T) {
			pkgPath := "./" + leaf
			if _, err := os.Stat(leaf); os.IsNotExist(err) {
				t.Fatalf("leaf %q listed in project-context.md Rule 19 but directory %s does not exist", leaf, leaf)
			}

			cmd := exec.Command(
				"go", "list", "-deps",
				"-f", "{{if not .Standard}}{{.ImportPath}}{{end}}",
				pkgPath,
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("go list -deps %s failed: %v\nstderr: %s", pkgPath, err, stderr.String())
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

// TestScanImports_DetectsViolation proves that the real enforcement helper
// (scanImports) actually flags bad imports in production files AND correctly
// ignores external test packages. Without this, the production TestXxx tests
// could pass vacuously if scanImports silently inspected nothing or if the
// external-test-package skip regressed.
//
// This replaces the previous tautological sanity check that re-implemented
// parser.ParseFile inline and therefore did not exercise scanImports at all.
func TestScanImports_DetectsViolation(t *testing.T) {
	tmp := t.TempDir()

	must := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Production file, no forbidden import — should be scanned but not flagged.
	must("good.go", "package x\nimport _ \"fmt\"\n")
	// Production file with the forbidden import — MUST be flagged.
	must("bad.go", "package x\nimport _ \""+importSubtitle+"\"\n")
	// External test package (name ends in _test). Same forbidden import, but
	// external test packages are a separate compilation unit and CAN import
	// peer/child packages without creating a cycle. MUST be skipped.
	must("x_test.go", "package x_test\nimport _ \""+importSubtitle+"\"\n")
	// A non-.go file in the same dir — must be ignored by the walker.
	must("README.md", "not a go file\n")

	violations, scanned, err := scanImports(tmp, importSubtitle)
	if err != nil {
		t.Fatalf("scanImports returned error: %v", err)
	}

	// scanned should count good.go + bad.go = 2 (README.md filtered by extension,
	// x_test.go filtered by external-test-pkg skip).
	if scanned != 2 {
		t.Errorf("expected scanned=2 (good.go + bad.go), got %d", scanned)
	}

	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation (bad.go), got %d: %v", len(violations), violations)
	}
	if filepath.Base(violations[0]) != "bad.go" {
		t.Errorf("expected bad.go to be flagged, got %s", violations[0])
	}
}

// scanImports walks every .go file directly under dirRel and returns the
// paths of files that import `forbidden`. It skips:
//   - directories (sub-packages are out of scope — Rule 19 talks about
//     direct-package boundaries; if sub-packages are ever added under
//     services/handlers/repository, revisit this)
//   - non-.go files
//   - files whose package name ends in "_test" (external test packages;
//     they are a separate compilation unit and may legitimately import
//     peer packages that the production code cannot)
//
// Returning violations (rather than calling t.Errorf directly) is what makes
// this helper testable on its own — see TestScanImports_DetectsViolation.
func scanImports(dirRel, forbidden string) (violations []string, scanned int, err error) {
	entries, err := os.ReadDir(dirRel)
	if err != nil {
		return nil, 0, err
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(dirRel, entry.Name())
		file, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return nil, 0, perr
		}
		if strings.HasSuffix(file.Name.Name, "_test") {
			continue
		}
		scanned++

		for _, imp := range file.Imports {
			if strings.Trim(imp.Path.Value, `"`) == forbidden {
				violations = append(violations, path)
			}
		}
	}
	return violations, scanned, nil
}

// assertNoImport is the t.Helper adapter around scanImports used by the
// production Rule-19 tests.
func assertNoImport(t *testing.T, dirRel, forbidden, why string) {
	t.Helper()

	violations, scanned, err := scanImports(dirRel, forbidden)
	if err != nil {
		t.Fatalf("scanImports(internal/%s) failed: %v", dirRel, err)
	}
	if scanned == 0 {
		t.Fatalf("no .go files scanned in internal/%s/ — wrong working directory?", dirRel)
	}
	for _, v := range violations {
		t.Errorf("%s imports %s — violates %s. See project-context.md Rule 19.", v, forbidden, why)
	}
}
