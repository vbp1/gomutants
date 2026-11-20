package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gomutants/gomutants/internal/coverage"
	"github.com/gomutants/gomutants/internal/discover"
)

type cacheDependencies struct {
	tests    map[string][]string
	packages map[string][]string
	coverage []string
}

// cacheDependencyFiles includes packages without tests and dependencies
// outside the mutation scope. Both can decide a covering test's verdict.
func cacheDependencyFiles(ctx context.Context, dir string, patterns []string, tags string, testFlags []string) (*cacheDependencies, error) {
	// This reads inputs, not executable metadata; VCS stamping is irrelevant.
	args := []string{"list", "-deps", "-test", "-json", "-buildvcs=false"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, coverage.BuildFlags(testFlags)...)
	args = append(args, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing cache dependencies: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	pkgs := make(map[string]discover.Package)
	var inputs []string
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var pkg struct {
			discover.Package
			CgoFiles       []string
			TestEmbedFiles []string
			Standard       bool
			ForTest        string
		}
		if err := decoder.Decode(&pkg); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decoding cache dependencies: %w", err)
		}
		// go list -test names generated test mains "<package>.test".
		// Their generated sources are not stable project inputs.
		if pkg.Standard || strings.HasSuffix(pkg.ImportPath, ".test") {
			continue
		}
		pkg.GoFiles = append(pkg.GoFiles, pkg.CgoFiles...)
		for _, file := range slices.Concat(pkg.GoFiles, pkg.EmbedFiles, pkg.TestEmbedFiles) {
			inputs = append(inputs, filepath.Join(pkg.Dir, file))
		}
		// Test variants contribute their test files above, but their
		// rewritten import paths must not replace the production graph.
		if pkg.ForTest == "" {
			pkgs[pkg.ImportPath] = pkg.Package
		}
	}
	production := packageDependencyFiles(pkgs, false)
	dependencies := &cacheDependencies{tests: packageDependencyFiles(pkgs, true), packages: make(map[string][]string), coverage: inputs}
	for name, pkg := range pkgs {
		dependencies.packages[pkg.Dir] = production[name]
	}
	return dependencies, nil
}

func packageDependencyFiles(pkgs map[string]discover.Package, includeTests bool) map[string][]string {
	result := make(map[string][]string, len(pkgs))
	for name, root := range pkgs {
		pending := slices.Clone(root.Imports)
		if includeTests {
			pending = append(pending, root.TestImports...)
			pending = append(pending, root.XTestImports...)
		}
		seen := map[string]bool{name: true}
		for len(pending) > 0 {
			dep := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if seen[dep] {
				continue
			}
			seen[dep] = true
			pkg := pkgs[dep]
			for _, file := range slices.Concat(pkg.GoFiles, pkg.EmbedFiles) {
				result[name] = append(result[name], filepath.Join(pkg.Dir, file))
			}
			// Only the root's tests run; imported packages' tests do not.
			pending = append(pending, pkg.Imports...)
		}
	}
	return result
}
