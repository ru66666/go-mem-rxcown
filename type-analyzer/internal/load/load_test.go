package load

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dby/type-analyzer/internal/config"
)

func TestVendorModeLoadsVendoredDependency(t *testing.T) {
	root, err := filepath.Abs("testdata/vendorproject")
	if err != nil {
		t.Fatal(err)
	}
	c := config.Config{}
	c.Target.Root, c.Target.Mode = root, "vendor"
	c.Target.Patterns = []string{"./..."}
	c.Target.Vendor.LegacyImportPath = "example.test/vendorproject"
	loaded, diags, err := Packages(c)
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Cleanup()
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diags)
	}
	found := false
	var paths []string
	for _, p := range loaded.Packages {
		paths = append(paths, p.PkgPath)
		if p.PkgPath == "example.test/vendorproject/vendor/example.test/dependency" {
			found = true
		}
	}
	if !found {
		t.Fatalf("vendored dependency was not loaded: %v", paths)
	}
}

func TestVendorModeReusesExistingGOPATH(t *testing.T) {
	workspace := t.TempDir()
	importPath := "example.test/vendorproject"
	root := filepath.Join(workspace, "src", filepath.FromSlash(importPath))
	if err := os.CopyFS(root, os.DirFS("testdata/vendorproject")); err != nil {
		t.Fatal(err)
	}
	if got, ok := gopathFor(root, importPath); !ok || got != workspace {
		t.Fatalf("gopathFor() = %q, %t; want %q, true", got, ok, workspace)
	}
	c := config.Config{}
	c.Target.Root, c.Target.Mode, c.Target.Patterns = root, "vendor", []string{"./..."}
	c.Target.Vendor.LegacyImportPath = importPath
	loaded, diags, err := Packages(c)
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Cleanup()
	if len(diags) != 0 || len(loaded.Packages) == 0 {
		t.Fatalf("packages=%d diagnostics=%#v", len(loaded.Packages), diags)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("existing GOPATH project was unexpectedly removed: %v", err)
	}
}
