package load

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dby/type-analyzer/internal/config"
	"github.com/dby/type-analyzer/internal/model"
	"golang.org/x/tools/go/packages"
)

type Result struct {
	Packages         []*packages.Package
	Mode, ImportPath string
	Cleanup          func()
}

func Packages(c config.Config) (Result, []model.Diagnostic, error) {
	mode := c.Target.Mode
	if mode == "auto" {
		if _, err := os.Stat(filepath.Join(c.Target.Root, "go.mod")); err == nil {
			mode = "module"
		} else {
			mode = "vendor"
		}
	}
	if mode == "vendor" {
		return vendorPackages(c)
	}
	return load(c, c.Target.Root, c.Target.Patterns, nil, "module", "", func() {})
}

func load(c config.Config, dir string, patterns, extraEnv []string, mode, importPath string, cleanup func()) (Result, []model.Diagnostic, error) {
	env := append([]string{}, os.Environ()...)
	for _, entry := range extraEnv {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env = setEnv(env, parts[0], parts[1])
		}
	}
	for k, v := range c.Target.Build.Env {
		env = setEnv(env, k, v)
	}
	var flags []string
	if len(c.Target.Build.Tags) > 0 {
		flags = []string{"-tags=" + strings.Join(c.Target.Build.Tags, ",")}
	}
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps | packages.NeedModule, Dir: dir, Env: env, BuildFlags: flags}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		cleanup()
		return Result{}, nil, err
	}
	seen := map[string]bool{}
	var all []*packages.Package
	var diags []model.Diagnostic
	var visit func(*packages.Package)
	visit = func(p *packages.Package) {
		if p == nil || seen[p.ID] {
			return
		}
		seen[p.ID] = true
		all = append(all, p)
		for _, e := range p.Errors {
			diags = append(diags, model.Diagnostic{Package: p.PkgPath, Position: e.Pos, Message: e.Msg})
		}
		for _, q := range p.Imports {
			visit(q)
		}
	}
	for _, p := range pkgs {
		visit(p)
	}
	if c.Analysis.Strict && len(diags) > 0 {
		cleanup()
		return Result{}, diags, fmt.Errorf("package loading reported %d error(s)", len(diags))
	}
	return Result{Packages: all, Mode: mode, ImportPath: importPath, Cleanup: cleanup}, diags, nil
}

func vendorPackages(c config.Config) (Result, []model.Diagnostic, error) {
	if c.Target.Vendor.LegacyImportPath == "" {
		return Result{}, nil, fmt.Errorf("target.vendor.legacy_import_path is required in vendor mode")
	}
	if _, err := os.Stat(filepath.Join(c.Target.Root, "vendor")); err != nil {
		return Result{}, nil, fmt.Errorf("vendor mode requires %s/vendor: %w", c.Target.Root, err)
	}
	patterns := vendorPatterns(c.Target.Patterns, c.Target.Vendor.LegacyImportPath)
	// A project already placed at $GOPATH/src/<import-path> needs neither a
	// copy nor a temporary directory. In particular this preserves large vendor
	// trees and lets callers intentionally maintain a classic GOPATH workspace.
	if gopath, ok := gopathFor(c.Target.Root, c.Target.Vendor.LegacyImportPath); ok {
		return load(c, c.Target.Root, patterns, []string{"GO111MODULE=off", "GOPATH=" + gopath}, "vendor", c.Target.Vendor.LegacyImportPath, func() {})
	}
	tmp, err := os.MkdirTemp("", "type-analyzer-gopath-")
	if err != nil {
		return Result{}, nil, err
	}
	link := filepath.Join(tmp, "src", filepath.FromSlash(c.Target.Vendor.LegacyImportPath))
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		os.RemoveAll(tmp)
		return Result{}, nil, err
	}
	// The Go command deliberately ignores symlinked directories when expanding
	// <import-path>/... in GOPATH mode. Copying is therefore necessary here.
	if err := os.CopyFS(link, os.DirFS(c.Target.Root)); err != nil {
		os.RemoveAll(tmp)
		return Result{}, nil, fmt.Errorf("copy project into temporary GOPATH: %w", err)
	}
	return load(c, link, patterns, []string{"GO111MODULE=off", "GOPATH=" + tmp}, "vendor", c.Target.Vendor.LegacyImportPath, func() { _ = os.RemoveAll(tmp) })
}

func vendorPatterns(patterns []string, importPath string) []string {
	result := make([]string, len(patterns))
	base := strings.TrimSuffix(importPath, "/")
	for i, p := range patterns {
		if p == "." || p == "./" {
			result[i] = base
		} else {
			result[i] = base + "/" + strings.TrimPrefix(p, "./")
		}
	}
	return result
}

// gopathFor returns the GOPATH root only when root is exactly
// $GOPATH/src/<legacy import path>. A mere directory named "src" elsewhere is
// insufficient because the complete import-path suffix is also verified.
func gopathFor(root, importPath string) (string, bool) {
	root = filepath.Clean(root)
	rel := filepath.Clean(filepath.FromSlash(importPath))
	if rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	src := root
	for range strings.Split(rel, string(filepath.Separator)) {
		src = filepath.Dir(src)
	}
	if filepath.Base(src) != "src" || filepath.Clean(filepath.Join(src, rel)) != root {
		return "", false
	}
	return filepath.Dir(src), true
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, x := range env {
		if strings.HasPrefix(x, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
