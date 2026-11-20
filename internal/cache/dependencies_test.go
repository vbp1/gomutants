package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasherPackageDependencies(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg")
	if err := os.Mkdir(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(pkg, "pkg.go"), "package pkg\n")
	dep := filepath.Join(dir, "dep.go")
	mustWrite(t, dep, "package dep\nconst Value = 1\n")
	h := NewHasher(nil)
	before, err := h.HashPkgFiles(pkg)
	if err != nil {
		t.Fatal(err)
	}
	h.SetPackageDependencies(map[string][]string{pkg: {dep}})
	after, err := h.HashPkgFiles(pkg)
	if err != nil || after == before {
		t.Fatalf("dependency must change memoized package hash: before=%s after=%s err=%v", before, after, err)
	}
	h.SetPackageDependencies(map[string][]string{pkg: {filepath.Join(dir, "missing.go")}})
	if _, err := h.HashPkgFiles(pkg); err == nil {
		t.Fatal("missing dependency must fail hashing")
	}
}

func TestHasherCoverageDependencies(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module deps\n\ngo 1.26\n")
	mustWrite(t, filepath.Join(dir, "pkg.go"), "package deps\n")
	dep := filepath.Join(t.TempDir(), "dep.go")
	mustWrite(t, dep, "package dep\nconst Value = 1\n")
	h := NewHasher(nil)
	before, err := h.HashCoverageInputs([]string{dir}, dir, "", "", "", "go1.26", "")
	if err != nil {
		t.Fatal(err)
	}
	h.SetCoverageDependencies([]string{dep})
	after, err := h.HashCoverageInputs([]string{dir}, dir, "", "", "", "go1.26", "")
	if err != nil || after == before {
		t.Fatalf("dependency must change coverage hash: before=%s after=%s err=%v", before, after, err)
	}
	h.SetCoverageDependencies([]string{filepath.Join(dir, "missing.go")})
	if _, err := h.HashCoverageInputs([]string{dir}, dir, "", "", "", "go1.26", ""); err == nil {
		t.Fatal("missing dependency must fail hashing")
	}
}
