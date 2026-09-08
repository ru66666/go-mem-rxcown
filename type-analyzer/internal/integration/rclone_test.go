//go:build !short

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dby/type-analyzer/internal/config"
	"github.com/dby/type-analyzer/internal/extract"
	"github.com/dby/type-analyzer/internal/load"
	"github.com/dby/type-analyzer/internal/model"
)

// TestRcloneModuleAndVendor runs the two real-project cases requested in README.
// It is intentionally opt-in: both fixtures are large and rclone's dependency graph is costly.
// Run: TYPE_ANALYZER_RCLONE=1 go test ./internal/integration -run TestRclone -v
func TestRcloneModuleAndVendor(t *testing.T) {
	if os.Getenv("TYPE_ANALYZER_RCLONE") != "1" {
		t.Skip("set TYPE_ANALYZER_RCLONE=1 to analyze rclone testdata")
	}
	root := filepath.Join("..", "..", "testdata")
	cases := []struct{ name, dir, mode string }{
		{"module", "rclone-module", "module"},
		{"vendor", "rclone-vendor", "vendor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project, err := filepath.Abs(filepath.Join(root, tc.dir))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(project); err != nil {
				t.Skipf("fixture missing: %v", err)
			}
			if tc.mode == "vendor" {
				if _, err := os.Stat(filepath.Join(project, "vendor", "modules.txt")); err != nil {
					t.Fatalf("invalid vendor fixture: %v", err)
				}
				if _, err := os.Stat(filepath.Join(project, "go.mod")); !os.IsNotExist(err) {
					t.Fatalf("vendor fixture must not contain go.mod")
				}
			}
			c := config.Config{}
			// fs contains interface and embedding samples; backend/local supplies
			// a concrete implementation and both value/pointer field references.
			c.Target.Root, c.Target.Mode, c.Target.Patterns = project, tc.mode, []string{"./fs", "./backend/local"}
			c.Target.Vendor.LegacyImportPath = "github.com/rclone/rclone"
			c.Analysis.IncludeDependencies = true
			c.Analysis.IncludeStdlib = false
			loaded, diags, err := load.Packages(c)
			if err != nil {
				t.Fatal(err)
			}
			defer loaded.Cleanup()
			if len(diags) != 0 {
				t.Fatalf("package diagnostics: %#v", diags)
			}
			g := extract.Graph(c, model.Run{ID: "rclone-" + tc.mode, ProjectRoot: project, LoadMode: tc.mode}, loaded.Packages, diags)
			if !containsType(g, "github.com/rclone/rclone/fs.Fs", "interface") {
				t.Fatalf("fs.Fs interface not found; extracted %d types", len(g.Types))
			}
			if len(g.Types) < 20 || len(g.Methods) == 0 {
				t.Fatalf("unexpectedly small graph: types=%d methods=%d", len(g.Types), len(g.Methods))
			}
			if len(g.Implementers) == 0 || len(g.Embeddings) == 0 || len(g.References) == 0 {
				t.Fatalf("required relations missing: implements=%d embeds=%d references=%d", len(g.Implementers), len(g.Embeddings), len(g.References))
			}
			assertImplementation(t, g, "github.com/rclone/rclone/backend/local.Fs", "github.com/rclone/rclone/fs.Fs", false, true)
			assertFieldRelation(t, g.Embeddings, "github.com/rclone/rclone/fs.OverrideRemote", "github.com/rclone/rclone/fs.ObjectInfo", "ObjectInfo", 0, "")
			assertFieldRelation(t, g.References, "github.com/rclone/rclone/backend/local.Fs", "github.com/rclone/rclone/fs.Features", "features", 1, "direct")
			assertFieldRelation(t, g.References, "github.com/rclone/rclone/backend/local.Fs", "github.com/rclone/rclone/backend/local.Options", "opt", 0, "direct")
		})
	}
}

func containsType(g model.Graph, id, kind string) bool {
	for _, typ := range g.Types {
		if typ.ID == id && typ.Kind == kind {
			return true
		}
	}
	return false
}

func assertImplementation(t *testing.T, g model.Graph, concreteID, interfaceID string, value, pointer bool) {
	t.Helper()
	for _, r := range g.Implementers {
		if r.ConcreteID == concreteID && r.InterfaceID == interfaceID {
			if r.ValueImpl != value || r.PointerImpl != pointer {
				t.Fatalf("implementation flags: %#v", r)
			}
			return
		}
	}
	t.Fatalf("implementation %s -> %s was not found", concreteID, interfaceID)
}

func assertFieldRelation(t *testing.T, relations []model.FieldRelation, sourceID, targetID, field string, pointerDepth int, referencePath string) {
	t.Helper()
	for _, r := range relations {
		if r.SourceID == sourceID && r.TargetID == targetID && r.FieldName == field && r.PointerDepth == pointerDepth && r.ReferencePath == referencePath {
			return
		}
	}
	t.Fatalf("relation %s.%s -> %s (pointer_depth=%d, path=%q) was not found", sourceID, field, targetID, pointerDepth, referencePath)
}
