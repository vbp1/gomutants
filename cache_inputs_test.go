package main

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gomutants/gomutants/internal/discover"
)

func TestPackageDependencyFiles(t *testing.T) {
	pkgs := map[string]discover.Package{
		"consumer": {Dir: "consumer", GoFiles: []string{"consumer.go"}, Imports: []string{"adapter"},
			TestImports: []string{"helper"}, XTestImports: []string{"external"}},
		"adapter": {Dir: "adapter", GoFiles: []string{"adapter.go"}, Imports: []string{"leaf", "settings"},
			TestImports: []string{"unrelated"}},
		"leaf":      {Dir: "leaf", GoFiles: []string{"leaf.go"}},
		"settings":  {Dir: "settings", GoFiles: []string{"settings.go"}, EmbedFiles: []string{"value.txt"}},
		"helper":    {Dir: "helper", GoFiles: []string{"helper.go"}},
		"external":  {Dir: "external", GoFiles: []string{"external.go"}},
		"unrelated": {Dir: "unrelated", GoFiles: []string{"unrelated.go"}},
	}
	got := packageDependencyFiles(pkgs, true)["consumer"]
	slices.Sort(got)
	want := []string{
		filepath.Join("adapter", "adapter.go"), filepath.Join("external", "external.go"),
		filepath.Join("helper", "helper.go"), filepath.Join("leaf", "leaf.go"),
		filepath.Join("settings", "settings.go"), filepath.Join("settings", "value.txt"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("dependency files = %v, want %v", got, want)
	}
	production := packageDependencyFiles(pkgs, false)["consumer"]
	slices.Sort(production)
	want = []string{filepath.Join("adapter", "adapter.go"), filepath.Join("leaf", "leaf.go"),
		filepath.Join("settings", "settings.go"), filepath.Join("settings", "value.txt")}
	if !slices.Equal(production, want) {
		t.Fatalf("production dependencies = %v, want %v", production, want)
	}
}

func TestCacheDependencyFilesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cacheDependencyFiles(ctx, t.TempDir(), []string{"./..."}, "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled dependency listing: %v, want context.Canceled", err)
	}
}
