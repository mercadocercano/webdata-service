package port

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TEST-ID: O2c guard (PLAT-E39 T9c2, condición vinculante de @dev-security)
//
// FindAllActiveByCategoryPlatform is a cross-tenant read with no tenant_id parameter and
// no RLS backstop on webdata_sources. If a non-admin handler ever calls it, it reopens
// exactly the cross-tenant bypass this epic exists to close. This guard asserts that the
// ONLY production (non-test) caller of the method is the enrichment usecase.
//
// Allowed references:
//   - this port declaration (source_repository.go)
//   - the repository implementation (postgres_source_repository.go)
//   - the sole allowed caller: product/application/usecase/enrich_global_products_usecase.go
//   - test fakes (any *_test.go) — excluded
//
// Any other production file referencing the method fails the build's safety contract.

func TestFindAllActiveByCategoryPlatform_OnlyEnrichmentUseCaseCallsIt(t *testing.T) {
	repoRoot := findRepoRoot(t)

	allowed := map[string]bool{
		filepath.Join(repoRoot, "src/source/domain/port/source_repository.go"):                              true, // declaration
		filepath.Join(repoRoot, "src/source/infrastructure/persistence/postgres_source_repository.go"):     true, // implementation
		filepath.Join(repoRoot, "src/product/application/usecase/enrich_global_products_usecase.go"):      true, // sole allowed caller
	}

	method := "FindAllActiveByCategoryPlatform"
	var offenders []string

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Exclude tests — fakes legitimately implement the interface.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if !referencesIdent(path, method) {
			return nil
		}
		if !allowed[path] {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking repo: %v", err)
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("FindAllActiveByCategoryPlatform referenced outside its allowed callers "+
			"(port decl, repo impl, enrichment usecase). Offenders:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// referencesIdent reports whether path's Go source mentions the given identifier as a
// selector or ident (cheap lexical check — sufficient for a guard; a non-allowed caller
// would have to spell the method name to call it).
func referencesIdent(path, name string) bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		// Parse error in a production file is itself a problem worth surfacing.
		return false
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	// This test file lives at <repo>/src/source/domain/port/. Walk up to the dir holding go.mod.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate repo root (go.mod) from " + file)
	return ""
}